package flaps

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

func Retry(op func() error) error {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 100 * time.Millisecond
	bo.MaxInterval = 500 * time.Millisecond
	bo.MaxElapsedTime = 0 // no stop
	bo.RandomizationFactor = 0.5
	bo.Multiplier = 2
	return backoff.Retry(op, bo)
}
