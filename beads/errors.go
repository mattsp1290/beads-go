package beads

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"strconv"
	"strings"
)

// Kind classifies an operation error.
type Kind string

const (
	KindNotFound    Kind = "not_found"
	KindValidation  Kind = "validation"
	KindTimeout     Kind = "timeout"
	KindTransport   Kind = "transport"
	KindAuthFailed  Kind = "auth_failed"
	KindUnsupported Kind = "unsupported"
	KindBadResponse Kind = "bad_response"
	KindExit        Kind = "exit"
)

// Error is the structured error envelope returned by this package.
type Error struct {
	Op     string
	Kind   Kind
	Status int
	Err    error
}

// Error returns a human-readable error string.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	parts := []string{"beads"}
	if e.Op != "" {
		parts = append(parts, e.Op)
	}
	if e.Kind != "" {
		parts = append(parts, string(e.Kind))
	}
	if e.Status != 0 {
		parts = append(parts, "status "+strconv.Itoa(e.Status))
	}
	msg := strings.Join(parts, ": ")
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}
	return msg
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Is reports whether e matches a package sentinel or another Error with the
// same non-empty Kind.
func (e *Error) Is(target error) bool {
	if e == nil || target == nil {
		return false
	}
	switch target {
	case ErrNotFound:
		return e.Kind == KindNotFound
	case ErrUnsupported:
		return e.Kind == KindUnsupported
	case ErrValidation:
		return e.Kind == KindValidation
	}
	var targetErr *Error
	if errors.As(target, &targetErr) && targetErr.Kind != "" {
		return e.Kind == targetErr.Kind
	}
	return false
}

var (
	ErrNotFound    error = kindSentinel{kind: KindNotFound, message: "beads: not found"}
	ErrUnsupported error = kindSentinel{kind: KindUnsupported, message: "beads: unsupported"}
	ErrValidation  error = kindSentinel{kind: KindValidation, message: "beads: validation"}
)

type kindSentinel struct {
	kind    Kind
	message string
}

func (s kindSentinel) Error() string {
	return s.message
}

func validationError(op, format string, args ...any) error {
	return &Error{
		Op:   op,
		Kind: KindValidation,
		Err:  fmt.Errorf(format, args...),
	}
}

func badResponseError(op string, err error) error {
	if err == nil {
		return nil
	}
	return &Error{
		Op:   op,
		Kind: KindBadResponse,
		Err:  err,
	}
}

func classifyExecError(op string, err error, stderr []byte, stdout ...[]byte) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &Error{Op: op, Kind: KindTimeout, Err: err}
	}

	var execErr *exec.Error
	if errors.As(err, &execErr) {
		return &Error{Op: op, Kind: KindTransport, Err: err}
	}
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return &Error{Op: op, Kind: KindTransport, Err: err}
	}

	output := append([]byte(nil), stderr...)
	for _, part := range stdout {
		output = append(output, part...)
	}
	outputText := strings.ToLower(string(output))
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		switch {
		case matchesAny(outputText, authPatterns):
			return &Error{Op: op, Kind: KindAuthFailed, Err: err}
		case matchesAny(outputText, notFoundPatterns):
			return &Error{Op: op, Kind: KindNotFound, Err: err}
		case matchesAny(outputText, validationPatterns):
			return &Error{Op: op, Kind: KindValidation, Err: err}
		default:
			return &Error{Op: op, Kind: KindExit, Status: exitErr.ExitCode(), Err: err}
		}
	}

	return &Error{Op: op, Kind: KindTransport, Err: err}
}

var authPatterns = []string{
	"permission denied",
	"unauthorized",
	"forbidden",
}

var notFoundPatterns = []string{
	"not found",
	"no issue found",
	"no such issue",
	"issue does not exist",
}

var validationPatterns = []string{
	"invalid status",
	"unknown status",
	"invalid priority",
	"validation failed",
}

func matchesAny(haystack string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}
