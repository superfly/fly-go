package flaps

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// Retry the given operation with exponential backoff up to 1 minute.
func Retry(ctx context.Context, op func() error) error {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 100 * time.Millisecond
	bo.MaxInterval = 500 * time.Millisecond
	bo.MaxElapsedTime = 1 * time.Minute
	bo.RandomizationFactor = 0.5
	bo.Multiplier = 2
	bo.Reset()
	return backoff.Retry(op, backoff.WithContext(bo, ctx))
}
