package cmd

import (
	"errors"

	"github.com/diffpal/diffpal/internal/reviewer"
)

type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *exitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (e *exitError) ExitCode() int {
	if e == nil || e.code <= 0 {
		return 1
	}
	return e.code
}

func withExitCode(code int, err error) error {
	if err == nil {
		return nil
	}
	var coder interface{ ExitCode() int }
	if errors.As(err, &coder) {
		return err
	}
	return &exitError{code: code, err: err}
}

func reviewExitError(err error) error {
	if err == nil {
		return nil
	}
	var reviewErr *reviewer.Error
	if errors.As(err, &reviewErr) {
		switch reviewErr.Kind {
		case reviewer.KindConfig:
			return withExitCode(2, err)
		case reviewer.KindTransient:
			return withExitCode(3, err)
		default:
			return withExitCode(5, err)
		}
	}
	return withExitCode(5, err)
}
