package anonsurf

import (
	"fmt"
	"strings"
)

type errorCollector struct {
	errors []error
}

func (c *errorCollector) add(operation string, err error) {
	if err == nil {
		return
	}
	c.errors = append(c.errors, fmt.Errorf("%s: %w", operation, err))
}

func (c errorCollector) err() error {
	switch len(c.errors) {
	case 0:
		return nil
	case 1:
		return c.errors[0]
	default:
		return combinedError{errors: c.errors}
	}
}

type combinedError struct {
	errors []error
}

func (e combinedError) Error() string {
	messages := make([]string, 0, len(e.errors))
	for _, err := range e.errors {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

func (e combinedError) Unwrap() error {
	return e.errors[0]
}

type transitionError struct {
	operation error
	rollback  error
}

func (e transitionError) Error() string {
	return fmt.Sprintf("%v; rollback failed: %v", e.operation, e.rollback)
}

func (e transitionError) Unwrap() error {
	return e.operation
}
