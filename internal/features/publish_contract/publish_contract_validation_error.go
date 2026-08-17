package publish_contract

// ValidationFailedError carries the violations the broker accumulated for a contract,
// so the command can render them instead of a single formatted line.
type ValidationFailedError struct {
	Message    string
	Violations []string
}

func (e *ValidationFailedError) Error() string {
	return e.Message
}
