package common

type LimitExceededError struct{}

func (e *LimitExceededError) Error() string { return "Month limit exceeded" }
