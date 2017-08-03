package concurrency

// RetryableError is an error which is retryable
type RetryableError struct {
	Err error
}

func (e RetryableError) Error() string { return e.Err.Error() }

// Retryable is true
func (e RetryableError) Retryable() bool { return true }

// NewRetryableError is a convenience to wrap another error in a retryable
func NewRetryableError(err error) error {
	return RetryableError{
		Err: err,
	}
}
