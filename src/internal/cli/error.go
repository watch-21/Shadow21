package cli

import "errors"

type actionableError struct {
	message string
	hint    string
}

func NewError(message, hint string) error {
	return actionableError{message: message, hint: hint}
}

func (e actionableError) Error() string {
	return e.message
}

func (e actionableError) Hint() string {
	return e.hint
}

func Hint(err error) string {
	var actionable interface {
		Hint() string
	}
	if errors.As(err, &actionable) {
		return actionable.Hint()
	}
	return ""
}
