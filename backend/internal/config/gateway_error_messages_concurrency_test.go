//go:build unit

package config

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestGatewayErrorMessageConcurrentSetAndRead is the regression test for the
// gatewayErrorMessagesLive lazy-pointer race.
//
// The old field was a *atomic.Pointer that SetGatewayErrorMessages lazily
// initialized with a plain (non-atomic) nil check and write. Concurrent setters
// racing on that initialization, or a getter reading the field while a setter
// writes it, was a data race on a raw pointer.
//
// The field is now a zero-value atomic.Pointer, so a raw &Config{} with no
// preinitialization must be safe to read and write from many goroutines at once.
// This test starts from exactly that state: every access is serialized through
// atomic Load/Store, and every observed outcome must be one of the valid
// fallback / static / live results.
func TestGatewayErrorMessageConcurrentSetAndRead(t *testing.T) {
	cfg := &Config{Gateway: GatewayConfig{ErrorMessages: map[string]string{
		"429": "static 429",
		"503": "static 503",
	}}}

	const (
		writerCount = 4
		readerCount = 4
		runWindow   = 200 * time.Millisecond
	)

	liveValues := make([]string, writerCount)
	liveSet := make(map[string]struct{}, writerCount)
	for w := 0; w < writerCount; w++ {
		liveValues[w] = fmt.Sprintf("live-%d", w)
		liveSet[liveValues[w]] = struct{}{}
	}

	stop := make(chan struct{})
	ready := make(chan struct{})
	var wg sync.WaitGroup

	for w := 0; w < writerCount; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			<-ready
			live := liveValues[w]
			for {
				select {
				case <-stop:
					return
				default:
				}
				switch w % 3 {
				case 0: // publish a live override
					cfg.SetGatewayErrorMessages(map[string]string{"429": live})
				case 1: // nil clears any override
					cfg.SetGatewayErrorMessages(nil)
				case 2: // empty map clears any override
					cfg.SetGatewayErrorMessages(map[string]string{})
				}
			}
		}(w)
	}

	for r := 0; r < readerCount; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ready
			for {
				select {
				case <-stop:
					return
				default:
				}
				// 429 is covered by static config and by every published live
				// snapshot: the answer must be the static value or a live value
				// some writer actually published — never fallback, never garbage.
				got := GatewayErrorMessage(cfg, 429, "fallback")
				if got != "static 429" {
					if _, ok := liveSet[got]; !ok {
						t.Errorf("GatewayErrorMessage(429) = %q; want \"static 429\" or a published live value", got)
						return
					}
				}
				// 503 exists only in static config: writers never publish it, so
				// the only valid outcome is the static value.
				if got := GatewayErrorMessage(cfg, 503, "fallback"); got != "static 503" {
					t.Errorf("GatewayErrorMessage(503) = %q; want static \"static 503\"", got)
					return
				}
				// 500 is configured nowhere: the only valid outcome is fallback.
				if got := GatewayErrorMessage(cfg, 500, "fallback"); got != "fallback" {
					t.Errorf("GatewayErrorMessage(500) = %q; want fallback", got)
					return
				}
				// The live getter must return nil (cleared) or a snapshot whose
				// single entry is a value some writer published.
				if live := cfg.GatewayErrorMessagesLive(); live != nil {
					if len(live) != 1 {
						t.Errorf("GatewayErrorMessagesLive() = %v; want nil or a single {429: live} snapshot", live)
						return
					}
					msg, ok := live["429"]
					if !ok {
						t.Errorf("GatewayErrorMessagesLive() = %v; missing 429 key", live)
						return
					}
					if _, ok := liveSet[msg]; !ok {
						t.Errorf("GatewayErrorMessagesLive() = %v; 429 value %q was never published", live, msg)
						return
					}
				}
			}
		}()
	}

	// Release all goroutines at once so the setters' first accesses (which is
	// where the old lazy-pointer initialization raced) overlap.
	close(ready)
	deadline := time.Now().Add(runWindow)
	for time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	close(stop)
	wg.Wait()
}
