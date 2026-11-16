package tokens

import (
	"sync"
	"testing"
)

// Parse builds the macaroon slice with append, so three macaroons leave it
// with spare capacity. normalized must not append into that capacity.
func TestNormalizedLeavesTokenSlicesUntouched(t *testing.T) {
	tok := Parse("FlyV1 fm2_a,fm2_b,fm2_c,oauth1")
	if cap(tok.macaroons) <= len(tok.macaroons) {
		t.Fatalf("test needs spare capacity, got len %d cap %d", len(tok.macaroons), cap(tok.macaroons))
	}

	if got, want := tok.GraphQL(), "fm2_a,fm2_b,fm2_c,oauth1"; got != want {
		t.Fatalf("GraphQL() = %q, want %q", got, want)
	}

	if spare := tok.macaroons[:cap(tok.macaroons)][len(tok.macaroons)]; spare != "" {
		t.Fatalf("GraphQL() wrote %q into the spare capacity of the macaroon slice", spare)
	}
}

func TestNormalizedConcurrentReaders(t *testing.T) {
	tok := Parse("FlyV1 fm2_a,fm2_b,fm2_c,oauth1")

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = tok.GraphQL()
				_ = tok.All()
				_ = tok.GraphQLHeader()
			}
		}()
	}
	wg.Wait()
}
