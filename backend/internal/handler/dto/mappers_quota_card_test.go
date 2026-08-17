package dto

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func quotaCardFixture() (*service.GroupQuotaCardResult, *service.Group) {
	rem5h := 62.5
	rem7d := 85.0
	resetsAt := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	result := &service.GroupQuotaCardResult{
		GroupID:          7,
		TotalRemaining5h: &rem5h,
		TotalRemaining7d: &rem7d,
		Accounts: []service.GroupQuotaAccount{
			{
				AccountID:   42,
				DisplayName: "prod-account",
				FiveHour:    &service.WindowUsage{Utilization: 38.0, ResetsAt: &resetsAt},
				SevenDay:    &service.WindowUsage{Utilization: 15.0, ResetsAt: nil},
			},
		},
	}
	group := &service.Group{ID: 7, Name: "claude-team", Platform: service.PlatformAnthropic}
	return result, group
}

func TestGroupQuotaCardFromService_mapsGroupAndAccountFields(t *testing.T) {
	// Given: a quota card result with one fully populated account
	result, group := quotaCardFixture()
	// When: mapping to the DTO
	out := GroupQuotaCardFromService(result, group)
	// Then: group identity, totals and account fields are carried over
	if out == nil {
		t.Fatal("expected non-nil DTO")
	}
	if out.GroupID != 7 || out.GroupName != "claude-team" || out.Platform != service.PlatformAnthropic {
		t.Fatalf("unexpected group fields: %+v", out)
	}
	if out.TotalRemaining5h == nil || *out.TotalRemaining5h != 62.5 {
		t.Fatalf("unexpected total_remaining_5h: %v", out.TotalRemaining5h)
	}
	if out.TotalRemaining7d == nil || *out.TotalRemaining7d != 85.0 {
		t.Fatalf("unexpected total_remaining_7d: %v", out.TotalRemaining7d)
	}
	if len(out.Accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(out.Accounts))
	}
	acct := out.Accounts[0]
	if acct.AccountID == nil || *acct.AccountID != 42 {
		t.Fatalf("expected account_id 42, got %v", acct.AccountID)
	}
	if acct.DisplayName != "prod-account" {
		t.Fatalf("unexpected display_name: %q", acct.DisplayName)
	}
	if acct.FiveHour == nil || acct.FiveHour.Utilization != 38.0 || acct.FiveHour.ResetsAt == nil {
		t.Fatalf("unexpected five_hour window: %+v", acct.FiveHour)
	}
	if acct.SevenDay == nil || acct.SevenDay.Utilization != 15.0 || acct.SevenDay.ResetsAt != nil {
		t.Fatalf("unexpected seven_day window: %+v", acct.SevenDay)
	}
}

func TestGroupQuotaCardFromService_whenAnonymized_thenAccountIDSerializesNull(t *testing.T) {
	// Given: an anonymized account entry (service uses AccountID 0 for 账号N)
	result, group := quotaCardFixture()
	result.Accounts[0].AccountID = 0
	result.Accounts[0].DisplayName = "账号1"
	// When: mapping and serializing
	out := GroupQuotaCardFromService(result, group)
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Then: account_id is present and explicitly null (frontend contract)
	if !strings.Contains(string(raw), `"account_id":null`) {
		t.Fatalf("expected \"account_id\":null in %s", raw)
	}
	if !strings.Contains(string(raw), `"display_name":"账号1"`) {
		t.Fatalf("expected anonymized display name in %s", raw)
	}
}

func TestGroupQuotaCardFromService_whenNoUsageData_thenTotalsAndWindowsSerializeNull(t *testing.T) {
	// Given: a result with no window data anywhere
	result := &service.GroupQuotaCardResult{
		GroupID: 7,
		Accounts: []service.GroupQuotaAccount{
			{AccountID: 9, DisplayName: "idle-account"},
		},
	}
	group := &service.Group{ID: 7, Name: "g", Platform: service.PlatformOpenAI}
	// When: mapping and serializing
	out := GroupQuotaCardFromService(result, group)
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Then: totals and both windows serialize as explicit null
	for _, want := range []string{`"total_remaining_5h":null`, `"total_remaining_7d":null`, `"five_hour":null`, `"seven_day":null`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("expected %s in %s", want, raw)
		}
	}
}

func TestGroupQuotaCardFromService_whenEmptyAccounts_thenAccountsIsEmptyArray(t *testing.T) {
	// Given: a group with no accounts
	result := &service.GroupQuotaCardResult{GroupID: 7}
	group := &service.Group{ID: 7, Name: "g", Platform: service.PlatformOpenAI}
	// When: mapping and serializing
	out := GroupQuotaCardFromService(result, group)
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Then: accounts serializes as [] rather than null
	if !strings.Contains(string(raw), `"accounts":[]`) {
		t.Fatalf("expected \"accounts\":[] in %s", raw)
	}
}

func TestGroupQuotaCardFromService_whenWindowStatsPresent_thenWindowStatsSerialized(t *testing.T) {
	// Given: an account whose 5h and 7d windows each carry window stats
	resetsAt := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	result := &service.GroupQuotaCardResult{
		GroupID: 7,
		Accounts: []service.GroupQuotaAccount{
			{
				AccountID:   42,
				DisplayName: "prod-account",
				FiveHour: &service.WindowUsage{
					Utilization: 38.0,
					ResetsAt:    &resetsAt,
					WindowStats: &service.WindowStats{Requests: 120, Tokens: 45000, Cost: 3.5, UserCost: 4.2},
				},
				SevenDay: &service.WindowUsage{
					Utilization: 15.0,
					WindowStats: &service.WindowStats{Requests: 900, Tokens: 310000, Cost: 27.75, UserCost: 31.1},
				},
			},
		},
	}
	group := &service.Group{ID: 7, Name: "claude-team", Platform: service.PlatformAnthropic}
	// When: mapping and serializing
	out := GroupQuotaCardFromService(result, group)
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Then: window_stats carries requests, tokens, cost and user_cost on both windows
	five := out.Accounts[0].FiveHour
	if five == nil || five.WindowStats == nil {
		t.Fatalf("expected five_hour window_stats, got %+v", out.Accounts[0])
	}
	if five.WindowStats.Requests != 120 || five.WindowStats.Tokens != 45000 || five.WindowStats.Cost != 3.5 {
		t.Fatalf("unexpected five_hour window_stats: %+v", five.WindowStats)
	}
	if five.WindowStats.UserCost == nil || *five.WindowStats.UserCost != 4.2 {
		t.Fatalf("unexpected five_hour user_cost: %v", five.WindowStats.UserCost)
	}
	seven := out.Accounts[0].SevenDay
	if seven == nil || seven.WindowStats == nil {
		t.Fatalf("expected seven_day window_stats, got %+v", out.Accounts[0])
	}
	if seven.WindowStats.Requests != 900 || seven.WindowStats.Tokens != 310000 || seven.WindowStats.Cost != 27.75 {
		t.Fatalf("unexpected seven_day window_stats: %+v", seven.WindowStats)
	}
	if seven.WindowStats.UserCost == nil || *seven.WindowStats.UserCost != 31.1 {
		t.Fatalf("unexpected seven_day user_cost: %v", seven.WindowStats.UserCost)
	}
	for _, want := range []string{
		`"five_hour":{"utilization":38,"resets_at":"2026-07-27T12:00:00Z","window_stats":{"requests":120,"tokens":45000,"cost":3.5,"user_cost":4.2}}`,
		`"seven_day":{"utilization":15,"resets_at":null,"window_stats":{"requests":900,"tokens":310000,"cost":27.75,"user_cost":31.1}}`,
	} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("expected %s in %s", want, raw)
		}
	}
}

func TestGroupQuotaCardFromService_whenNilInput_thenNil(t *testing.T) {
	// Given: nil inputs
	// When/Then: mapper returns nil instead of panicking
	if GroupQuotaCardFromService(nil, &service.Group{}) != nil {
		t.Fatal("expected nil for nil result")
	}
	if GroupQuotaCardFromService(&service.GroupQuotaCardResult{}, nil) != nil {
		t.Fatal("expected nil for nil group")
	}
}

func TestGroupViewGrantFromService_mapsUsernamesAndTimestamp(t *testing.T) {
	// Given: a grant record enriched with usernames
	grantedAt := time.Date(2026, 7, 26, 8, 30, 0, 0, time.UTC)
	grant := &service.GroupViewGrantWithUser{
		GroupViewGrant:    service.GroupViewGrant{UserID: 11, GrantedAt: grantedAt},
		Username:          "alice",
		Email:             "alice@test.com",
		GrantedByUsername: "admin-bob",
	}
	// When: mapping and serializing
	out := GroupViewGrantFromService(grant)
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Then: user_id, username, email, granted_by and RFC3339 granted_at are present
	if out.UserID != 11 || out.Username != "alice" || out.Email != "alice@test.com" || out.GrantedBy != "admin-bob" {
		t.Fatalf("unexpected entry: %+v", out)
	}
	if !strings.Contains(string(raw), `"granted_at":"2026-07-26T08:30:00Z"`) {
		t.Fatalf("expected RFC3339 granted_at in %s", raw)
	}
}

func TestGroupViewGrantCandidateFromService_mapsUserAndGrantedFlag(t *testing.T) {
	// Given: a candidate user that already holds a grant on the group being edited
	u := &service.User{
		ID:       7,
		Username: "carol",
		Email:    "carol@test.com",
		Role:     service.RoleUser,
		Status:   service.StatusActive,
	}
	// When: mapping with granted=true
	out := GroupViewGrantCandidateFromService(u, true)
	// Then: identity fields carry over and the granted flag is preserved
	if out.UserID != 7 || out.Username != "carol" || out.Email != "carol@test.com" {
		t.Fatalf("unexpected candidate: %+v", out)
	}
	if out.Role != service.RoleUser || out.Status != service.StatusActive || !out.Granted {
		t.Fatalf("unexpected candidate: %+v", out)
	}
	// And: a nil user maps to nil rather than a zero-value entry
	if GroupViewGrantCandidateFromService(nil, false) != nil {
		t.Fatal("expected nil for nil user")
	}
}

func TestViewableGroupFromService_mapsPinnedFields(t *testing.T) {
	// Given: a viewable group
	g := &service.ViewableGroup{GroupID: 3, GroupName: "codex-team", Platform: service.PlatformOpenAI, Status: service.StatusActive}
	// When: mapping and serializing
	out := ViewableGroupFromService(g)
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Then: exactly the pinned fields are serialized (no status)
	if out.GroupID != 3 || out.GroupName != "codex-team" || out.Platform != service.PlatformOpenAI {
		t.Fatalf("unexpected entry: %+v", out)
	}
	if strings.Contains(string(raw), "status") {
		t.Fatalf("status must not leak into the response: %s", raw)
	}
}
