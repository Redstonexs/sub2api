package service

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/mdhtml"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	// announcementBroadcastWorkers is the number of goroutines sending broadcast emails.
	announcementBroadcastWorkers = 3
	// announcementBroadcastBuffer is the size of the pending-send channel.
	announcementBroadcastBuffer = 256
	// announcementBroadcastUserPage is the user pagination size for recipient resolution.
	announcementBroadcastUserPage = 500
	// announcementBroadcastSendTimeout bounds a single recipient send.
	announcementBroadcastSendTimeout = 30 * time.Second
	// announcementBroadcastListTimeout bounds a single page of user listing.
	announcementBroadcastListTimeout = 30 * time.Second
)

// announcementBroadcastJob is a single rendered email to deliver to one recipient.
type announcementBroadcastJob struct {
	announcementID int64
	title          string
	contentHTML    string
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
			"announcement_title": job.title,
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
	targeting := ann.Targeting

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.LegacyPrintf("service.announcement_broadcast",
					"[AnnouncementBroadcast] dispatch panic for announcement %d: %v", annID, r)
			}
		}()
		s.resolveAndEnqueue(annID, title, contentHTML, targeting)
	}()
}

// resolveAndEnqueue pages through all users, matches the targeting rules, and enqueues
// one send job per matching recipient. It applies backpressure: when the worker queue
// is full it blocks until a slot frees up (or shutdown), so recipients are never dropped.
func (s *AnnouncementBroadcastService) resolveAndEnqueue(annID int64, title, contentHTML string, targeting AnnouncementTargeting) {
	now := time.Now()
	enqueued := 0
	suppressed := 0

	for page := 1; ; page++ {
		select {
		case <-s.stopCh:
			return
		default:
		}

		listCtx, cancel := context.WithTimeout(context.Background(), announcementBroadcastListTimeout)
		users, result, err := s.userRepo.List(listCtx, pagination.PaginationParams{
			Page:     page,
			PageSize: announcementBroadcastUserPage,
		})
		cancel()
		if err != nil {
			logger.LegacyPrintf("service.announcement_broadcast",
				"[AnnouncementBroadcast] list users (page %d) failed for announcement %d: %v", page, annID, err)
			return
		}

		for i := range users {
			u := users[i]
			email := strings.TrimSpace(u.Email)
			if email == "" {
				continue
			}
			if !targeting.Matches(u.Balance, activeSubscriptionGroupIDs(u.Subscriptions, now)) {
				continue
			}

			name := strings.TrimSpace(u.Username)
			if name == "" {
				name = emailRecipientName(email)
			}

			unsubscribeCtx, unsubscribeCancel := context.WithTimeout(context.Background(), announcementBroadcastListTimeout)
			unsubscribed, err := s.notificationEmailService.IsUnsubscribed(unsubscribeCtx, email, NotificationEmailEventAnnouncementBroadcast)
			unsubscribeCancel()
			if err != nil {
				logger.LegacyPrintf("service.announcement_broadcast",
					"[AnnouncementBroadcast] unsubscribe lookup failed for announcement %d recipient %s: %v", annID, email, err)
				continue
			}
			if unsubscribed {
				suppressed++
				continue
			}

			job := announcementBroadcastJob{
				announcementID: annID,
				title:          title,
				contentHTML:    contentHTML,
				userID:         u.ID,
				email:          email,
				name:           name,
			}
			select {
			case s.jobs <- job:
				enqueued++
			case <-s.stopCh:
				return
			}
		}

		if result == nil || page >= result.Pages || len(users) == 0 {
			break
		}
	}

	logger.LegacyPrintf("service.announcement_broadcast",
		"[AnnouncementBroadcast] enqueued %d recipients, suppressed %d recipients for announcement %d", enqueued, suppressed, annID)
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
