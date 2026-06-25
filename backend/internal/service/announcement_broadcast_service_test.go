package service

import (
	"testing"
	"time"
)

func TestActiveSubscriptionGroupIDsFiltersExpiredAndDedupes(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	subs := []UserSubscription{
		{GroupID: 1, ExpiresAt: now.Add(time.Hour)},     // active
		{GroupID: 2, ExpiresAt: now.Add(-time.Hour)},    // expired -> excluded
		{GroupID: 1, ExpiresAt: now.Add(2 * time.Hour)}, // duplicate group -> deduped
		{GroupID: 3, ExpiresAt: now.Add(time.Minute)},   // active
	}

	got := activeSubscriptionGroupIDs(subs, now)

	if len(got) != 2 {
		t.Fatalf("expected 2 active groups, got %d (%v)", len(got), got)
	}
	if _, ok := got[1]; !ok {
		t.Fatalf("expected group 1 present, got %v", got)
	}
	if _, ok := got[3]; !ok {
		t.Fatalf("expected group 3 present, got %v", got)
	}
	if _, ok := got[2]; ok {
		t.Fatalf("expected expired group 2 to be excluded, got %v", got)
	}
}

func TestActiveSubscriptionGroupIDsEmpty(t *testing.T) {
	if got := activeSubscriptionGroupIDs(nil, time.Now()); got != nil {
		t.Fatalf("expected nil for no subscriptions, got %v", got)
	}
}
