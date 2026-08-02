package service

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/mdhtml"
)

const (
	// announcementBroadcastWorkers is the number of goroutines sending broadcast emails.
	announcementBroadcastWorkers = 3
	// announcementBroadcastBuffer is the size of the pending-send channel.
	announcementBroadcastBuffer = 256
	// announcementBroadcastSendTimeout bounds a single recipient send.
	// Recipient paging and its timeout now live in announcement_audience.go, which
	// the admin audience preview shares.
	announcementBroadcastSendTimeout = 30 * time.Second
)

// announcementBroadcastJob is a single rendered email to deliver to one recipient.
type announcementBroadcastJob struct {
	announcementID int64
	title          string
	contentHTML    string
	severity       string
	userID         int64
	email          string
	name           string
}

// AnnouncementBroadcastService fans out an announcement to every targeted user as an
// email, using an in-memory worker pool. Delivery is made idempotent/resume-safe by
// NotificationEmailService's per-recipient delivery key, so re-publishing an
// announcement or restarting the process never double-sends.
type AnnouncementBroadcastService struct {
	userRepo                 UserRepository
	notificationEmailService *NotificationEmailService

	jobs     chan announcementBroadcastJob
	wg       sync.WaitGroup
	stopCh   chan struct{}
	stopOnce sync.Once
	workers  int
}

// NewAnnouncementBroadcastService creates the service and starts its worker pool.
func NewAnnouncementBroadcastService(userRepo UserRepository, notificationEmailService *NotificationEmailService) *AnnouncementBroadcastService {
	s := &AnnouncementBroadcastService{
		userRepo:                 userRepo,
		notificationEmailService: notificationEmailService,
		jobs:                     make(chan announcementBroadcastJob, announcementBroadcastBuffer),
		stopCh:                   make(chan struct{}),
		workers:                  announcementBroadcastWorkers,
	}
	s.start()
	return s
}

func (s *AnnouncementBroadcastService) start() {
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}
	logger.LegacyPrintf("service.announcement_broadcast", "[AnnouncementBroadcast] Started %d workers", s.workers)
}

func (s *AnnouncementBroadcastService) worker(id int) {
	defer s.wg.Done()
	for {
		select {
		case job := <-s.jobs:
			s.processJob(id, job)
		case <-s.stopCh:
			return
		}
	}
}

func (s *AnnouncementBroadcastService) processJob(workerID int, job announcementBroadcastJob) {
	if s.notificationEmailService == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), announcementBroadcastSendTimeout)
	defer cancel()

	// The severity label is localized per recipient, so it is resolved here rather
	// than once in Dispatch.
	severityLabel := announcementSeverityLabel(
		job.severity,
		s.notificationEmailService.ResolveRecipientLocale(ctx, job.userID, job.email),
	)

	err := s.notificationEmailService.Send(ctx, NotificationEmailSendInput{
		Event:          NotificationEmailEventAnnouncementBroadcast,
		RecipientEmail: job.email,
		RecipientName:  job.name,
		UserID:         job.userID,
		// SourceType+SourceID scope the delivery dedup key to this announcement, so each
		// recipient is emailed at most once per announcement (resume-safe across restarts).
		SourceType: "announcement",
		SourceID:   strconv.FormatInt(job.announcementID, 10),
		Variables: map[string]string{
			"announcement_title":          job.title,
			"announcement_severity_label": severityLabel,
		},
		// announcement_content is pre-escaped safe HTML (see mdhtml.ToSafeHTML) and is
		// injected raw so paragraph/line breaks render instead of being escaped again.
		RawHTMLVariables: map[string]string{
			"announcement_content": job.contentHTML,
		},
	})
	if err != nil {
		logger.LegacyPrintf("service.announcement_broadcast",
			"[AnnouncementBroadcast] worker %d failed to send announcement %d to %s: %v", workerID, job.announcementID, job.email, err)
	}
}

// Dispatch asynchronously emails an announcement to every user matching its targeting.
// It returns immediately; recipient resolution and sending happen in the background.
// Callers should only invoke this for active announcements whose notify mode is email.
func (s *AnnouncementBroadcastService) Dispatch(ann *Announcement) {
	if s == nil || ann == nil || s.notificationEmailService == nil {
		return
	}

	annID := ann.ID
	title := ann.Title
	contentHTML := mdhtml.ToSafeHTML(ann.Content)
	severity := ann.Severity
	targeting := ann.Targeting

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.LegacyPrintf("service.announcement_broadcast",
					"[AnnouncementBroadcast] dispatch panic for announcement %d: %v", annID, r)
			}
		}()
		s.resolveAndEnqueue(annID, title, contentHTML, severity, targeting)
	}()
}

// resolveAndEnqueue scans the audience and enqueues one send job per deliverable
// recipient. It applies backpressure: when the worker queue is full it blocks until
// a slot frees up (or shutdown), so recipients are never dropped.
//
// The scan itself lives in announcement_audience.go and is shared with the admin's
// audience preview, so the count shown before publishing is the count that is sent.
func (s *AnnouncementBroadcastService) resolveAndEnqueue(annID int64, title, contentHTML, severity string, targeting AnnouncementTargeting) {
	aborted := false
	stats, err := scanAnnouncementAudience(
		context.Background(), s.userRepo, s.notificationEmailService,
		targeting, 0, s.stopCh,
		func(recipient AnnouncementRecipient) bool {
			job := announcementBroadcastJob{
				announcementID: annID,
				title:          title,
				contentHTML:    contentHTML,
				severity:       severity,
				userID:         recipient.UserID,
				email:          recipient.Email,
				name:           recipient.Name,
			}
			select {
			case s.jobs <- job:
				return true
			case <-s.stopCh:
				aborted = true
				return false
			}
		},
	)
	if err != nil {
		logger.LegacyPrintf("service.announcement_broadcast",
			"[AnnouncementBroadcast] audience scan failed for announcement %d: %v", annID, err)
		return
	}
	if aborted {
		logger.LegacyPrintf("service.announcement_broadcast",
			"[AnnouncementBroadcast] shutdown interrupted announcement %d after %d recipients", annID, stats.Deliverable)
		return
	}

	logger.LegacyPrintf("service.announcement_broadcast",
		"[AnnouncementBroadcast] enqueued %d recipients, suppressed %d recipients for announcement %d",
		stats.Deliverable, stats.Unsubscribed, annID)
}

// SendTest delivers one announcement email to a single recipient so an admin can
// see the rendered result before publishing.
func (s *AnnouncementBroadcastService) SendTest(ctx context.Context, ann *Announcement, email, name string, userID int64) error {
	if s == nil || ann == nil || s.notificationEmailService == nil {
		return ErrAnnouncementTestEmailUnavailable
	}
	recipient := strings.TrimSpace(email)
	if recipient == "" {
		return ErrAnnouncementTestEmailUnavailable
	}

	// Send silently no-ops for an unsubscribed recipient on an optional event, so
	// without this check the admin would see a success and never get the mail.
	unsubscribed, err := s.notificationEmailService.IsUnsubscribed(ctx, recipient, NotificationEmailEventAnnouncementBroadcast)
	if err != nil {
		return err
	}
	if unsubscribed {
		return ErrAnnouncementTestEmailUnsubscribed
	}

	displayName := strings.TrimSpace(name)
	if displayName == "" {
		displayName = emailRecipientName(recipient)
	}
	locale := s.notificationEmailService.ResolveRecipientLocale(ctx, userID, recipient)

	return s.notificationEmailService.Send(ctx, NotificationEmailSendInput{
		Event:          NotificationEmailEventAnnouncementBroadcast,
		RecipientEmail: recipient,
		RecipientName:  displayName,
		UserID:         userID,
		// A distinct SourceType keeps test sends out of the real broadcast's dedup
		// namespace; the nonce ReminderKey additionally keeps repeated tests to the
		// same admin from deduping against each other.
		SourceType:  "announcement_test",
		SourceID:    strconv.FormatInt(ann.ID, 10),
		ReminderKey: strconv.FormatInt(time.Now().UnixNano(), 10),
		Variables: map[string]string{
			"announcement_title":          ann.Title,
			"announcement_severity_label": announcementSeverityLabel(ann.Severity, locale),
		},
		RawHTMLVariables: map[string]string{
			"announcement_content": mdhtml.ToSafeHTML(ann.Content),
		},
	})
}

// Stop stops the worker pool and waits for in-flight sends to finish.
func (s *AnnouncementBroadcastService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
		s.wg.Wait()
		logger.LegacyPrintf("service.announcement_broadcast", "%s", "[AnnouncementBroadcast] All workers stopped")
	})
}

// activeSubscriptionGroupIDs returns the set of group IDs for the user's currently
// active (status=active and not expired) subscriptions. Users from UserRepository.List
// already have active-status subscriptions eager-loaded; we additionally drop expired
// ones to match UserSubscriptionRepository.ListActiveByUserID semantics.
func activeSubscriptionGroupIDs(subs []UserSubscription, now time.Time) map[int64]struct{} {
	if len(subs) == 0 {
		return nil
	}
	ids := make(map[int64]struct{}, len(subs))
	for i := range subs {
		if subs[i].ExpiresAt.After(now) {
			ids[subs[i].GroupID] = struct{}{}
		}
	}
	return ids
}

// announcementSeverityLabel renders a severity as short display text for emails.
// A label rather than a colour: notificationEmailCard bakes its header accent in at
// definition time, and an admin-customised stored template freezes it, so a colour
// placeholder would be silently ignored in exactly the templates that matter.
func announcementSeverityLabel(severity, locale string) string {
	chinese := strings.EqualFold(strings.TrimSpace(locale), notificationEmailLocaleChinese)
	switch severity {
	case AnnouncementSeverityCritical:
		if chinese {
			return "紧急"
		}
		return "Critical"
	case AnnouncementSeverityWarning:
		if chinese {
			return "重要"
		}
		return "Important"
	default:
		if chinese {
			return "通知"
		}
		return "Notice"
	}
}
