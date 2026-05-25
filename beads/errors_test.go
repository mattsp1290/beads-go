package beads

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestErrorString(t *testing.T) {
	t.Parallel()

	err := &Error{
		Op:     "Close",
		Kind:   KindExit,
		Status: 7,
		Err:    errors.New("bd failed"),
	}
	got := err.Error()
	for _, want := range []string{"beads", "Close", "exit", "status 7", "bd failed"} {
		if !strings.Contains(got, want) {
			t.Errorf("Error() = %q, missing %q", got, want)
		}
	}
}

func TestErrorUnwrap(t *testing.T) {
	t.Parallel()

	underlying := errors.New("root")
	err := &Error{Kind: KindTransport, Err: underlying}
	if !errors.Is(err, underlying) {
		t.Fatalf("errors.Is did not see underlying error")
	}
}

func TestErrorIsSentinels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		err    error
		target error
	}{
		{"not found", &Error{Kind: KindNotFound}, ErrNotFound},
		{"unsupported", &Error{Kind: KindUnsupported}, ErrUnsupported},
		{"validation", &Error{Kind: KindValidation}, ErrValidation},
		{"same kind", &Error{Kind: KindTimeout}, &Error{Kind: KindTimeout}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !errors.Is(tc.err, tc.target) {
				t.Fatalf("errors.Is(%v, %v) = false, want true", tc.err, tc.target)
			}
		})
	}
}

func TestErrorIsDoesNotMatchDifferentKind(t *testing.T) {
	t.Parallel()

	err := &Error{Kind: KindValidation}
	if errors.Is(err, ErrNotFound) {
		t.Fatal("validation matched ErrNotFound")
	}
	if errors.Is(err, &Error{Kind: KindTimeout}) {
		t.Fatal("validation matched timeout kind")
	}
}

func TestValidationErrorHelper(t *testing.T) {
	t.Parallel()

	err := validationError("Show", "bad %s", "id")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("validationError did not match ErrValidation: %v", err)
	}
	var be *Error
	if !errors.As(err, &be) {
		t.Fatalf("errors.As = false")
	}
	if be.Op != "Show" || be.Kind != KindValidation {
		t.Errorf("Error = %+v", be)
	}
}

func TestBadResponseError(t *testing.T) {
	t.Parallel()

	err := badResponseError("Ready", errors.New("decode"))
	var be *Error
	if !errors.As(err, &be) {
		t.Fatalf("errors.As = false")
	}
	if be.Op != "Ready" || be.Kind != KindBadResponse {
		t.Errorf("Error = %+v", be)
	}
}

func TestClassifyExecError(t *testing.T) {
	t.Parallel()

	exitErr := replayExitError(t, 7)
	cases := []struct {
		name   string
		err    error
		stderr []byte
		kind   Kind
		status int
	}{
		{"deadline", context.DeadlineExceeded, nil, KindTimeout, 0},
		{"canceled", context.Canceled, nil, KindTimeout, 0},
		{"exec error", &exec.Error{Name: "bd", Err: exec.ErrNotFound}, nil, KindTransport, 0},
		{"auth", exitErr, []byte("permission denied\n"), KindAuthFailed, 0},
		{"not found", exitErr, []byte("issue nope not found\n"), KindNotFound, 0},
		{"validation", exitErr, []byte("invalid status: nope\n"), KindValidation, 0},
		{"exit", exitErr, []byte("boom\n"), KindExit, 7},
		{"unknown non-exit", errors.New("network down"), nil, KindTransport, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := classifyExecError("Show", tc.err, tc.stderr)
			var be *Error
			if !errors.As(err, &be) {
				t.Fatalf("errors.As = false for %T: %v", err, err)
			}
			if be.Op != "Show" {
				t.Errorf("Op = %q", be.Op)
			}
			if be.Kind != tc.kind {
				t.Errorf("Kind = %q, want %q", be.Kind, tc.kind)
			}
			if be.Status != tc.status {
				t.Errorf("Status = %d, want %d", be.Status, tc.status)
			}
		})
	}
}

func TestClassifyExecErrorUsesStdoutJSONErrors(t *testing.T) {
	t.Parallel()

	err := classifyExecError("Transition", replayExitError(t, 1), nil, []byte(`{
		"error": "invalid status \"not_a_status\""
	}`))
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestClassifyExecErrorNil(t *testing.T) {
	t.Parallel()

	if err := classifyExecError("Ready", nil, nil); err != nil {
		t.Fatalf("classifyExecError(nil) = %v, want nil", err)
	}
}
