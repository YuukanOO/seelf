package bus

const (
	ErrorPolicyRetry              ErrorPolicy = iota // Default policy, retry the job later
	ErrorPolicyRetryPreserveOrder                    // Retry the job later but preserve the order of related jobs
	ErrorPolicyIgnore                                // Ignore the error and mark the job as done
)

type (
	ErrorPolicy uint8

	Error struct {
		policy ErrorPolicy
		err    error
	}
)

func (e Error) Error() string { return e.err.Error() }
func (e Error) Unwrap() error { return e.err }

// Wrap the given error with a job policy of ErrorPolicyIgnore which will mark
// the job as done and ignore the error if runned through the scheduler.
// If the error is nil, nil will be returned instead.
func Ignore(err error) error {
	if err == nil {
		return nil
	}

	return Error{
		policy: ErrorPolicyIgnore,
		err:    err,
	}
}

// Wrap the given error with a job policy of ErrorPolicyRetryPreserveOrder which will retry
// the job later but preserve the order of related jobs (ie. same dedupe name).
// If the error is nil, nil will be returned instead.
func PreserveOrder(err error) error {
	if err == nil {
		return nil
	}

	return Error{
		policy: ErrorPolicyRetryPreserveOrder,
		err:    err,
	}
}
