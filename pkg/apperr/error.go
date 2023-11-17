package apperr

import (
	"errors"
	"fmt"
)

var ErrNotFound = New("not_found") // Common error used when a resource could not be found.

// Represents an application error with an optional detail.
// Application errors represent an expected error from the domain perspective.
// Infrastructure errors should use the standard errors package.
type Error struct {
	Code   string `json:"code"`
	Detail error  `json:"detail,omitempty"`
}

// Instantiates a new RuleError with the given code.
func New(code string) error {
	return Error{code, nil}
}

// Instantiates a new RuleError error which wrap the detail error.
func NewWithDetail(code string, detailErr error) error {
	return Error{code, detailErr}
}

func (e Error) Error() string {
	if e.Detail == nil {
		return e.Code
	}

	return fmt.Sprintf("%s:%s", e.Code, e.Detail)
}

func (e Error) Unwrap() error { return e.Detail }

func (e Error) Is(err error) bool {
	derr, ok := err.(Error)

	if !ok {
		return false
	}

	return derr.Code == e.Code
}

// Wrap the given detail error into the base err.
// If err is an RuleError, its Detail field will be populated, else
// a new RuleError will be created.
func Wrap(err error, detail error) error {
	derr, ok := err.(Error)

	if !ok {
		return NewWithDetail(err.Error(), detail)
	}

	derr.Detail = detail

	return derr
}

// Same as errors.As but with generics =D
func As[T error](err error) (T, bool) {
	var target T
	ok := errors.As(err, &target)
	return target, ok
}
