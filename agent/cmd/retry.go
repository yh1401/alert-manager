package cmd

import (
	"log"
	"math/rand"
	"time"
)

// RetryableFunc defines a function signature for operations that can be retried.
// It should return `true` for `shouldRetry` if the error is transient and a retry is warranted.
// If the error is permanent, it should return `false`.
type RetryableFunc func() (shouldRetry bool, err error)

// WithRetry executes the provided function `fn` with a retry mechanism.
// It attempts the operation up to `attempts` times, with an exponential backoff delay between failures.
func WithRetry(attempts int, initialBackoff time.Duration, fn RetryableFunc) error {
	var err error
	var shouldRetry bool

	// Seed the random number generator to ensure different jitter values across agent restarts.
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < attempts; i++ {
		shouldRetry, err = fn()
		if err == nil {
			return nil // Operation was successful.
		}

		// If the error is determined to be non-retryable, stop immediately.
		if !shouldRetry {
			return err
		}

		// Stop if this was the last attempt.
		if i == attempts-1 {
			break
		}

		// Calculate exponential backoff: initialBackoff * 2^i
		backoff := initialBackoff * time.Duration(1<<i)

		// Add random jitter to prevent thundering herd: +/- up to 30% of backoff duration.
		jitter := time.Duration(r.Int63n(int64(backoff)*6/10) - int64(backoff)*3/10)
		waitTime := backoff + jitter
		if waitTime < 0 {
			waitTime = 0
		}

		log.Printf("⚠️ Operation failed, will retry in %v (attempt %d/%d). Error: %v", waitTime, i+1, attempts, err)
		time.Sleep(waitTime)
	}

	return err // All attempts failed; return the last error encountered.
}
