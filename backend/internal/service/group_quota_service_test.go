//go:build unit

package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Fake implementations for testing
// ---------------------------------------------------------------------------

type fakeGroupAccountRepo struct {
	AccountRepository // 嵌入接口——未实现的调用方方法会 panic（测试只调用 GetByIDs/ListByGroup）

	accountsByGroup map[int64][]*Account
	accountsByID    map[int64]*Account
}

func newFakeGroupAccountRepo() *fakeGroupAccountRepo {
	return &fakeGroupAccountRepo{
		accountsByGroup: make(map[int64][]*Account),
		accountsByID:    make(map[int64]*Account),
	}
}

func (r *fakeGroupAccountRepo) addAccount(groupID int64, a *Account) {
	r.accountsByGroup[groupID] = append(r.accountsByGroup[groupID], a)
	r.accountsByID[a.ID] = a
}

func (r *fakeGroupAccountRepo) ListByGroup(_ context.Context, groupID int64) ([]Account, error) {
	accs := r.accountsByGroup[groupID]
	result := make([]Account, 0, len(accs))
	for _, a := range accs {
		result = append(result, *a)
	}
	return result, nil
}

func (r *fakeGroupAccountRepo) GetByIDs(_ context.Context, ids []int64) ([]*Account, error) {
	result := make([]*Account, 0, len(ids))
	for _, id := range ids {
		if a, ok := r.accountsByID[id]; ok {
			result = append(result, a)
		}
	}
	return result, nil
}

type fakePassiveUsageBatchReader struct {
	usage map[int64]*UsageInfo
}

func newFakePassiveUsageBatchReader() *fakePassiveUsageBatchReader {
	return &fakePassiveUsageBatchReader{
		usage: make(map[int64]*UsageInfo),
	}
}

func (r *fakePassiveUsageBatchReader) GetPassiveUsageBatch(_ context.Context, accountIDs []int64) (map[int64]*UsageInfo, error) {
	result := make(map[int64]*UsageInfo)
	for _, id := range accountIDs {
		if u, ok := r.usage[id]; ok {
			result[id] = u
		}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeAnthropicOAuth(id int64) *Account {
	return &Account{
		ID:       id,
		Platform: PlatformAnthropic,
		Type:     AccountTypeOAuth,
		Name:     fmt.Sprintf("Test OAuth %d", id),
	}
}

func makeAnthropicSetupToken(id int64, sessionEnd time.Time, utilization float64) *Account {
	a := &Account{
		ID:                  id,
		Platform:            PlatformAnthropic,
		Type:                AccountTypeSetupToken,
		Name:                fmt.Sprintf("Test Setup %d", id),
		SessionWindowEnd:    &sessionEnd,
		SessionWindowStatus: "allowed_warning",
	}
	if utilization > 0 {
		a.Extra = map[string]any{
			"session_window_utilization": utilization,
		}
	}
	return a
}

func makeOpenAIOAuth(id int64) *Account {
	return &Account{
		ID:       id,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Name:     fmt.Sprintf("Test OpenAI %d", id),
	}
}

func makeCodexUsage5h(utilPercent float64, resetsAt time.Time) *UsageInfo {
	return &UsageInfo{
		UpdatedAt: &resetsAt,
		FiveHour: &UsageProgress{
			Utilization:      utilPercent,
			ResetsAt:         &resetsAt,
			RemainingSeconds: max(0, int(time.Until(resetsAt).Seconds())),
		},
	}
}

func makeCodexUsage7d(utilPercent float64, resetsAt time.Time) *UsageInfo {
	return &UsageInfo{
		UpdatedAt: &resetsAt,
		SevenDay: &UsageProgress{
			Utilization:      utilPercent,
			ResetsAt:         &resetsAt,
			RemainingSeconds: max(0, int(time.Until(resetsAt).Seconds())),
		},
	}
}

func makeCodexUsage5h7d(util5h, util7d float64, reset5h, reset7d time.Time) *UsageInfo {
	return &UsageInfo{
		UpdatedAt: &reset5h,
		FiveHour: &UsageProgress{
			Utilization:      util5h,
			ResetsAt:         &reset5h,
			RemainingSeconds: max(0, int(time.Until(reset5h).Seconds())),
		},
		SevenDay: &UsageProgress{
			Utilization:      util7d,
			ResetsAt:         &reset7d,
			RemainingSeconds: max(0, int(time.Until(reset7d).Seconds())),
		},
	}
}

func makeUsageWithBoth(util5h, util7d float64, reset5h, reset7d time.Time) *UsageInfo {
	return makeCodexUsage5h7d(util5h, util7d, reset5h, reset7d)
}

// ---------------------------------------------------------------------------
// GetGroupQuotaCard tests
// ---------------------------------------------------------------------------

func TestGetGroupQuotaCard_Aggregation_TwoWithData_OneWithout(t *testing.T) {
	// Given: 3 accounts, 2 with 5h usage data (util 80→rem 20, util 60→rem 40), 1 without
	repo := newFakeGroupAccountRepo()
	now := time.Now().UTC()
	reset := now.Add(3 * time.Hour)

	a1 := makeAnthropicSetupToken(1, reset, 0)
	a2 := makeAnthropicSetupToken(2, reset, 0)
	a3 := makeAnthropicSetupToken(3, reset, 0) // no usage data
	for _, a := range []*Account{a1, a2, a3} {
		repo.addAccount(1, a)
	}

	reader := newFakePassiveUsageBatchReader()
	reader.usage[1] = makeCodexUsage5h(80, reset) // → remaining 20
	reader.usage[2] = makeCodexUsage5h(60, reset) // → remaining 40
	// account 3 has no usage entry

	svc := NewGroupQuotaService(repo, reader)

	// When
	result, err := svc.GetGroupQuotaCard(context.Background(), 1, SortWindow5h, false)
	require.NoError(t, err)

	// Then: total remaining = mean of remaining over accounts WITH data = (20 + 40) / 2 = 30
	require.NotNil(t, result.TotalRemaining5h)
	assert.InDelta(t, 30.0, *result.TotalRemaining5h, 0.01)
	// 7d has no data → nil
	assert.Nil(t, result.TotalRemaining7d)
}

func TestGetGroupQuotaCard_Aggregation_AllLackData(t *testing.T) {
	// Given: all accounts lack usage data
	repo := newFakeGroupAccountRepo()
	a1 := makeAnthropicOAuth(1)
	a2 := makeAnthropicOAuth(2)
	for _, a := range []*Account{a1, a2} {
		repo.addAccount(1, a)
	}

	reader := newFakePassiveUsageBatchReader()
	// no usage entries

	svc := NewGroupQuotaService(repo, reader)

	// When
	result, err := svc.GetGroupQuotaCard(context.Background(), 1, SortWindow5h, false)
	require.NoError(t, err)

	// Then: total is nil when no account has that window's data
	assert.Nil(t, result.TotalRemaining5h)
	assert.Nil(t, result.TotalRemaining7d)
}

func TestGetGroupQuotaCard_Anonymization_SortIDs(t *testing.T) {
	// Given: accounts with IDs [5, 2, 8], all with 5h usage data
	repo := newFakeGroupAccountRepo()
	now := time.Now().UTC()
	reset := now.Add(3 * time.Hour)

	// Create in arbitrary order
	for _, id := range []int64{5, 2, 8} {
		a := makeAnthropicSetupToken(id, reset, 0)
		repo.addAccount(1, a)
	}

	reader := newFakePassiveUsageBatchReader()
	for _, id := range []int64{5, 2, 8} {
		reader.usage[id] = makeCodexUsage5h(50, reset)
	}

	svc := NewGroupQuotaService(repo, reader)

	// When
	result, err := svc.GetGroupQuotaCard(context.Background(), 1, SortWindow5h, true)
	require.NoError(t, err)

	// Then: display names assigned by ascending account ID
	// ID order: 2, 5, 8 → 账号1, 账号2, 账号3
	require.Len(t, result.Accounts, 3)
	assert.Equal(t, "账号1", result.Accounts[0].DisplayName)
	assert.Equal(t, "账号2", result.Accounts[1].DisplayName)
	assert.Equal(t, "账号3", result.Accounts[2].DisplayName)
	// AccountID should be 0 when anonymized
	assert.Zero(t, result.Accounts[0].AccountID)
	assert.Zero(t, result.Accounts[1].AccountID)
	assert.Zero(t, result.Accounts[2].AccountID)
}

func TestGetGroupQuotaCard_Anonymization_StableMapping(t *testing.T) {
	// Given: same accounts, first sorted by 5h, then by 7d
	repo := newFakeGroupAccountRepo()
	now := time.Now().UTC()
	reset := now.Add(3 * time.Hour)

	for _, id := range []int64{5, 2, 8} {
		a := makeAnthropicSetupToken(id, reset, 0)
		repo.addAccount(1, a)
	}

	reader := newFakePassiveUsageBatchReader()
	reader.usage[2] = makeUsageWithBoth(20, 80, reset, reset) // 5h rem=80, 7d rem=20
	reader.usage[5] = makeUsageWithBoth(90, 10, reset, reset) // 5h rem=10, 7d rem=90
	reader.usage[8] = makeUsageWithBoth(50, 50, reset, reset) // 5h rem=50, 7d rem=50

	svc := NewGroupQuotaService(repo, reader)

	// When: first query by 5h, then by 7d
	result5h, err := svc.GetGroupQuotaCard(context.Background(), 1, SortWindow5h, true)
	require.NoError(t, err)
	result7d, err := svc.GetGroupQuotaCard(context.Background(), 1, SortWindow7d, true)
	require.NoError(t, err)

	// Then: 账号N mapping is stable regardless of sort window.
	// ID 2 → 账号1, ID 5 → 账号2, ID 8 → 账号3 (always by ascending ID).
	// The order in Accounts differs by sort, but data attached to each 账号N is stable.
	extractMapping := func(r *GroupQuotaCardResult) map[string]*GroupQuotaAccount {
		m := make(map[string]*GroupQuotaAccount, len(r.Accounts))
		for i := range r.Accounts {
			a := &r.Accounts[i]
			m[a.DisplayName] = a
		}
		return m
	}
	m5h := extractMapping(result5h)
	m7d := extractMapping(result7d)

	// Verify mapping is as expected: ascending ID → ascending N
	assert.Len(t, m5h, 3)
	assert.Len(t, m7d, 3)

	// 账号1 (ID=2): 5h util=20 → rem=80, 7d util=80 → rem=20
	require.NotNil(t, m5h["账号1"])
	require.NotNil(t, m5h["账号1"].FiveHour)
	assert.InDelta(t, 20.0, m5h["账号1"].FiveHour.Utilization, 0.01)
	require.NotNil(t, m5h["账号1"].SevenDay)
	assert.InDelta(t, 80.0, m5h["账号1"].SevenDay.Utilization, 0.01)

	// 账号2 (ID=5): 5h util=90 → rem=10, 7d util=10 → rem=90
	require.NotNil(t, m5h["账号2"])
	require.NotNil(t, m5h["账号2"].FiveHour)
	assert.InDelta(t, 90.0, m5h["账号2"].FiveHour.Utilization, 0.01)
	require.NotNil(t, m5h["账号2"].SevenDay)
	assert.InDelta(t, 10.0, m5h["账号2"].SevenDay.Utilization, 0.01)

	// 账号3 (ID=8): 5h util=50 → rem=50, 7d util=50 → rem=50
	require.NotNil(t, m5h["账号3"])
	require.NotNil(t, m5h["账号3"].FiveHour)
	assert.InDelta(t, 50.0, m5h["账号3"].FiveHour.Utilization, 0.01)

	// Same data appears in 7d result — mapping is stable
	assert.Equal(t, m5h["账号1"].FiveHour.Utilization, m7d["账号1"].FiveHour.Utilization)
	assert.Equal(t, m5h["账号2"].FiveHour.Utilization, m7d["账号2"].FiveHour.Utilization)
	assert.Equal(t, m5h["账号3"].FiveHour.Utilization, m7d["账号3"].FiveHour.Utilization)
}

func TestGetGroupQuotaCard_Sort5h_DescWithNilLast(t *testing.T) {
	// Given: 3 accounts with 5h remaining [20, 40, nil-data] → order [40, 20, nil-last]
	repo := newFakeGroupAccountRepo()
	now := time.Now().UTC()
	reset := now.Add(3 * time.Hour)

	a1 := makeAnthropicSetupToken(1, reset, 0)
	a2 := makeAnthropicSetupToken(2, reset, 0)
	a3 := makeAnthropicSetupToken(3, reset, 0)
	for _, a := range []*Account{a1, a2, a3} {
		repo.addAccount(1, a)
	}

	reader := newFakePassiveUsageBatchReader()
	reader.usage[1] = makeCodexUsage5h(80, reset) // remaining = 20
	reader.usage[2] = makeCodexUsage5h(60, reset) // remaining = 40
	// account 3 has no data → nil remaining

	svc := NewGroupQuotaService(repo, reader)

	// When
	result, err := svc.GetGroupQuotaCard(context.Background(), 1, SortWindow5h, false)
	require.NoError(t, err)

	// Then: sorted by 5h remaining desc, nil last
	require.Len(t, result.Accounts, 3)
	assert.Equal(t, int64(2), result.Accounts[0].AccountID) // rem=40 (highest)
	assert.Equal(t, int64(1), result.Accounts[1].AccountID) // rem=20
	assert.Equal(t, int64(3), result.Accounts[2].AccountID) // nil-last
}

func TestGetGroupQuotaCard_Sort7d_SameSemantics(t *testing.T) {
	// Given: accounts with 7d data in various states
	repo := newFakeGroupAccountRepo()
	now := time.Now().UTC()
	reset := now.Add(7 * 24 * time.Hour)

	a1 := makeAnthropicSetupToken(1, reset, 0)
	a2 := makeAnthropicSetupToken(2, reset, 0)
	a3 := makeAnthropicSetupToken(3, reset, 0)
	for _, a := range []*Account{a1, a2, a3} {
		repo.addAccount(1, a)
	}

	reader := newFakePassiveUsageBatchReader()
	reader.usage[1] = makeCodexUsage7d(70, reset) // remaining = 30
	reader.usage[2] = makeCodexUsage7d(10, reset) // remaining = 90
	// account 3 has no 7d data

	svc := NewGroupQuotaService(repo, reader)

	// When
	result, err := svc.GetGroupQuotaCard(context.Background(), 1, SortWindow7d, false)
	require.NoError(t, err)

	// Then: sorted by 7d remaining desc, nil last
	require.Len(t, result.Accounts, 3)
	assert.Equal(t, int64(2), result.Accounts[0].AccountID) // rem=90 (highest)
	assert.Equal(t, int64(1), result.Accounts[1].AccountID) // rem=30
	assert.Equal(t, int64(3), result.Accounts[2].AccountID) // nil-last
}

func TestGetGroupQuotaCard_NoData_TieBreakByID_ASC(t *testing.T) {
	// Given: two no-data accounts [id=8, id=3]
	repo := newFakeGroupAccountRepo()
	a3 := makeAnthropicOAuth(3)
	a8 := makeAnthropicOAuth(8)
	// Add in reverse order to ensure repos don't pre-sort
	for _, a := range []*Account{a8, a3} {
		repo.addAccount(1, a)
	}

	reader := newFakePassiveUsageBatchReader()
	// no usage data

	svc := NewGroupQuotaService(repo, reader)

	// When
	result, err := svc.GetGroupQuotaCard(context.Background(), 1, SortWindow5h, false)
	require.NoError(t, err)

	// Then: tie-break by ID ascending → [3, 8]
	require.Len(t, result.Accounts, 2)
	assert.Equal(t, int64(3), result.Accounts[0].AccountID)
	assert.Equal(t, int64(8), result.Accounts[1].AccountID)
}

func TestGetGroupQuotaCard_NotAnonymized_ExposesRealIdentity(t *testing.T) {
	// Given: anonymize=false
	repo := newFakeGroupAccountRepo()
	now := time.Now().UTC()
	reset := now.Add(3 * time.Hour)

	a1 := makeAnthropicSetupToken(42, reset, 0)
	repo.addAccount(1, a1)

	reader := newFakePassiveUsageBatchReader()
	reader.usage[42] = makeCodexUsage5h(50, reset)

	svc := NewGroupQuotaService(repo, reader)

	// When
	result, err := svc.GetGroupQuotaCard(context.Background(), 1, SortWindow5h, false)
	require.NoError(t, err)

	// Then: real identity exposed
	require.Len(t, result.Accounts, 1)
	assert.Equal(t, int64(42), result.Accounts[0].AccountID)
	assert.Equal(t, "Test Setup 42", result.Accounts[0].DisplayName)
}

func TestGetGroupQuotaCard_EmptyGroup(t *testing.T) {
	// Given: a group with no accounts
	repo := newFakeGroupAccountRepo()
	reader := newFakePassiveUsageBatchReader()

	svc := NewGroupQuotaService(repo, reader)

	// When
	result, err := svc.GetGroupQuotaCard(context.Background(), 1, SortWindow5h, false)
	require.NoError(t, err)

	// Then: empty results, nil totals
	assert.Nil(t, result.TotalRemaining5h)
	assert.Nil(t, result.TotalRemaining7d)
	assert.Empty(t, result.Accounts)
	assert.Equal(t, int64(1), result.GroupID)
}

// ---------------------------------------------------------------------------
// GetPassiveUsageBatch tests
// ---------------------------------------------------------------------------

func TestGetPassiveUsageBatch_AnthropicOAuth(t *testing.T) {
	// Given: an Anthropic OAuth account with passive usage data in Extra
	now := time.Now().UTC()
	reset := now.Add(2 * time.Hour)
	acc := &Account{
		ID:       1,
		Platform: PlatformAnthropic,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			"passive_usage_sampled_at":     now.Format(time.RFC3339),
			"session_window_utilization":   0.35,
			"passive_usage_7d_utilization": 0.70,
			"passive_usage_7d_reset":       float64(reset.Unix()),
		},
		SessionWindowEnd: &reset,
	}

	repo := newFakeGroupAccountRepo()
	repo.addAccount(0, acc)
	svc := &AccountUsageService{accountRepo: repo}

	// When
	result, err := svc.GetPassiveUsageBatch(context.Background(), []int64{1})
	require.NoError(t, err)

	// Then: FiveHour and SevenDay populated from passive data
	require.Contains(t, result, int64(1))
	info := result[1]
	require.NotNil(t, info.FiveHour)
	assert.InDelta(t, 35.0, info.FiveHour.Utilization, 0.01)
	require.NotNil(t, info.SevenDay)
	assert.InDelta(t, 70.0, info.SevenDay.Utilization, 0.01)
	assert.Equal(t, "passive", info.Source)
}

func TestGetPassiveUsageBatch_AnthropicSetupToken(t *testing.T) {
	// Given: an Anthropic SetupToken account with session window
	reset := time.Now().UTC().Add(3 * time.Hour)
	acc := &Account{
		ID:                  2,
		Platform:            PlatformAnthropic,
		Type:                AccountTypeSetupToken,
		SessionWindowEnd:    &reset,
		SessionWindowStatus: "rejected",
	}

	repo := newFakeGroupAccountRepo()
	repo.addAccount(0, acc)
	svc := &AccountUsageService{accountRepo: repo}

	// When
	result, err := svc.GetPassiveUsageBatch(context.Background(), []int64{2})
	require.NoError(t, err)

	// Then: FiveHour populated from session window (rejected → 100%)
	require.Contains(t, result, int64(2))
	info := result[2]
	require.NotNil(t, info.FiveHour)
	assert.InDelta(t, 100.0, info.FiveHour.Utilization, 0.01)
	assert.Nil(t, info.SevenDay) // SetupToken can't get 7d
}

func TestGetPassiveUsageBatch_OpenAIOAuth_CodexKeys(t *testing.T) {
	// Given: an OpenAI OAuth account with codex Extra keys
	now := time.Now().UTC()
	reset5h := now.Add(4*time.Hour + 30*time.Minute) // future reset
	acc := &Account{
		ID:       3,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			"codex_5h_used_percent":        45.0,
			"codex_5h_reset_after_seconds": float64(16200),
			"codex_5h_reset_at":            reset5h.Format(time.RFC3339),
			"codex_7d_used_percent":        80.0,
			"codex_7d_reset_after_seconds": float64(432000),
			"codex_usage_updated_at":       now.Format(time.RFC3339),
		},
	}

	repo := newFakeGroupAccountRepo()
	repo.addAccount(0, acc)
	svc := &AccountUsageService{accountRepo: repo}

	// When
	result, err := svc.GetPassiveUsageBatch(context.Background(), []int64{3})
	require.NoError(t, err)

	// Then: FiveHour and SevenDay built from codex Extra data
	require.Contains(t, result, int64(3))
	info := result[3]
	require.NotNil(t, info.FiveHour)
	assert.InDelta(t, 45.0, info.FiveHour.Utilization, 0.01)
	assert.NotNil(t, info.FiveHour.ResetsAt)
	require.NotNil(t, info.SevenDay)
	assert.InDelta(t, 80.0, info.SevenDay.Utilization, 0.01)
	assert.NotNil(t, info.SevenDay.ResetsAt)
}

func TestGetPassiveUsageBatch_NonMatchingPlatform(t *testing.T) {
	// Given: a Gemini account (not Anthropic/OpenAI OAuth/SetupToken)
	acc := &Account{
		ID:       4,
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Extra:    map[string]any{},
	}

	repo := newFakeGroupAccountRepo()
	repo.addAccount(0, acc)
	svc := &AccountUsageService{accountRepo: repo}

	// When
	result, err := svc.GetPassiveUsageBatch(context.Background(), []int64{4})
	require.NoError(t, err)

	// Then: no entry for non-matching platform
	assert.NotContains(t, result, int64(4))
}

func TestGetPassiveUsageBatch_MultipleAccounts(t *testing.T) {
	// Given: 3 accounts of different types
	now := time.Now().UTC()
	reset := now.Add(3 * time.Hour)

	anthropicAcc := &Account{
		ID:       10,
		Platform: PlatformAnthropic,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			"session_window_utilization": 0.25,
		},
		SessionWindowEnd: &reset,
	}

	openAIAcc := &Account{
		ID:       20,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			"codex_5h_used_percent": 60.0,
			"codex_7d_used_percent": 30.0,
		},
	}

	geminiAcc := &Account{
		ID:       30,
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
	}

	repo := newFakeGroupAccountRepo()
	repo.addAccount(0, anthropicAcc)
	repo.addAccount(0, openAIAcc)
	repo.addAccount(0, geminiAcc)
	svc := &AccountUsageService{accountRepo: repo}

	// When
	result, err := svc.GetPassiveUsageBatch(context.Background(), []int64{10, 20, 30})
	require.NoError(t, err)

	// Then: Anthropic + OpenAI present, Gemini absent
	assert.Contains(t, result, int64(10))
	assert.Contains(t, result, int64(20))
	assert.NotContains(t, result, int64(30))

	// Verify Anthropic data
	assert.InDelta(t, 25.0, result[10].FiveHour.Utilization, 0.01)

	// Verify OpenAI data
	assert.InDelta(t, 60.0, result[20].FiveHour.Utilization, 0.01)
	assert.InDelta(t, 30.0, result[20].SevenDay.Utilization, 0.01)
}

func TestGetPassiveUsageBatch_EmptyInput(t *testing.T) {
	repo := newFakeGroupAccountRepo()
	svc := &AccountUsageService{accountRepo: repo}

	result, err := svc.GetPassiveUsageBatch(context.Background(), []int64{})
	require.NoError(t, err)
	assert.Empty(t, result)
}
