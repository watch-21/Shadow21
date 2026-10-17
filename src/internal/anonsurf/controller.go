package anonsurf

import (
	"errors"
)

var ErrTorrcNotFound = errors.New("torrc not found")

type StatusReporter func(string)

type Controller struct {
	statusReporter StatusReporter
}

func NewController(statusReporter StatusReporter) Controller {
	return Controller{statusReporter: statusReporter}
}

func (c Controller) reportStatus(status string) {
	if c.statusReporter != nil {
		c.statusReporter(status)
	}
}
