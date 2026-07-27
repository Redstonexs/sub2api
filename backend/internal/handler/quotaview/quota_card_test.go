package quotaview

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestParseSortWindow_whenEmpty_thenDefaultsToFiveHour(t *testing.T) {
	// Given: no sort query parameter
	// When: parsing
	got, ok := ParseSortWindow("")
	// Then: default 5h window, accepted
	if !ok {
		t.Fatal("expected empty sort to be accepted")
	}
	if got != service.SortWindow5h {
		t.Fatalf("expected default %q, got %q", service.SortWindow5h, got)
	}
}

func TestParseSortWindow_whenValid_thenMapsToServiceConstant(t *testing.T) {
	for _, raw := range []string{"5h", "7d"} {
		// Given: a valid sort value
		// When: parsing
		got, ok := ParseSortWindow(raw)
		// Then: accepted and mapped to the same-named service constant
		if !ok {
			t.Fatalf("expected sort %q to be accepted", raw)
		}
		if got != service.SortWindow(raw) {
			t.Fatalf("expected %q, got %q", service.SortWindow(raw), got)
		}
	}
}

func TestParseSortWindow_whenInvalid_thenRejected(t *testing.T) {
	for _, raw := range []string{"24h", "1d", "5H", "7D", " 5h", "5h ", "all"} {
		// Given: an invalid sort value
		// When: parsing
		_, ok := ParseSortWindow(raw)
		// Then: rejected so the handler can return 400
		if ok {
			t.Fatalf("expected sort %q to be rejected", raw)
		}
	}
}

func TestSupportsQuotaCard_platformGate(t *testing.T) {
	cases := []struct {
		platform string
		want     bool
	}{
		{service.PlatformAnthropic, true},
		{service.PlatformOpenAI, true},
		{service.PlatformGemini, false},
		{service.PlatformAntigravity, false},
		{service.PlatformGrok, false},
		{service.PlatformComposite, false},
		{"", false},
		{"unknown", false},
	}
	for _, tc := range cases {
		// Given: a group platform
		// When: checking quota card support
		got := SupportsQuotaCard(tc.platform)
		// Then: only anthropic/openai are supported
		if got != tc.want {
			t.Fatalf("SupportsQuotaCard(%q) = %v, want %v", tc.platform, got, tc.want)
		}
	}
}
