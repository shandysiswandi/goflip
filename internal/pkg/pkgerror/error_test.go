package pkgerror

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestTypeString(t *testing.T) {
	if got := TypeValidation.String(); got != "ERROR_TYPE_VALIDATION" {
		t.Fatalf("unexpected validation string: %q", got)
	}
	if got := TypeBusiness.String(); got != "ERROR_TYPE_BUSINESS" {
		t.Fatalf("unexpected business string: %q", got)
	}
	if got := TypeServer.String(); got != "ERROR_TYPE_SERVER" {
		t.Fatalf("unexpected server string: %q", got)
	}
	if got := Type(99).String(); got != "ERROR_TYPE_UNKNOWN" {
		t.Fatalf("unexpected unknown type string: %q", got)
	}
}

func TestCodeString(t *testing.T) {
	if got := CodeInvalidFormat.String(); got != "ERROR_CODE_INVALID_FORMAT" {
		t.Fatalf("unexpected invalid format string: %q", got)
	}
	if got := CodeConflict.String(); got != "ERROR_CODE_CONFLICT" {
		t.Fatalf("unexpected conflict string: %q", got)
	}
	if got := CodeInternal.String(); got != "ERROR_CODE_INTERNAL" {
		t.Fatalf("unexpected internal string: %q", got)
	}
	if got := Code(99).String(); got != "ERROR_CODE_INTERNAL" {
		t.Fatalf("unexpected default code string: %q", got)
	}
}

func TestErrorHelpers(t *testing.T) {
	root := errors.New("boom")
	err := NewServer(root)
	gerr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if !errors.Is(err, root) {
		t.Fatalf("expected wrapped error")
	}
	if got := gerr.Msg(); got != "Internal server error" {
		t.Fatalf("unexpected msg: %q", got)
	}
	if got := gerr.Type(); got != TypeServer {
		t.Fatalf("unexpected type: %v", got)
	}
	if got := gerr.Code(); got != CodeInternal {
		t.Fatalf("unexpected code: %v", got)
	}
	if got := gerr.Error(); got != "boom" {
		t.Fatalf("unexpected error string: %q", got)
	}
	if got := gerr.StatusCode(); got != http.StatusInternalServerError {
		t.Fatalf("unexpected status: %d", got)
	}
}

func TestBusinessAndValidationErrors(t *testing.T) {
	biz := NewBusiness("conflict", CodeConflict).(*Error)
	if got := biz.Error(); got != "conflict" {
		t.Fatalf("unexpected business error: %q", got)
	}
	if got := biz.StatusCode(); got != http.StatusConflict {
		t.Fatalf("unexpected business status: %d", got)
	}

	root := errors.New("bad")
	invalidInput := NewInvalidInput(root)
	if got := invalidInput.Error(); got != "bad" {
		t.Fatalf("unexpected invalid input error: %q", got)
	}
	if !errors.Is(invalidInput, root) {
		t.Fatalf("expected invalid input to wrap error")
	}

	invalidFormat := NewInvalidFormat().(*Error)
	if got := invalidFormat.Error(); got != "invalid request body" {
		t.Fatalf("unexpected invalid format error: %q", got)
	}
	if got := invalidFormat.StatusCode(); got != http.StatusBadRequest {
		t.Fatalf("unexpected invalid format status: %d", got)
	}
}

func TestErrorFallbackMessages(t *testing.T) {
	validation := new(nil, "", TypeValidation, CodeInternal).(*Error)
	if got := validation.Error(); got != "Validation violation" {
		t.Fatalf("unexpected validation fallback: %q", got)
	}

	business := new(nil, "", TypeBusiness, CodeInternal).(*Error)
	if got := business.Error(); got != "Logical business not meet with requirement" {
		t.Fatalf("unexpected business fallback: %q", got)
	}

	server := new(nil, "", TypeServer, CodeInternal).(*Error)
	if got := server.Error(); got != "Internal error" {
		t.Fatalf("unexpected server fallback: %q", got)
	}
}

func TestErrorStringIncludesDetails(t *testing.T) {
	err := NewBusiness("message", CodeForbidden).(*Error)
	str := err.String()
	if !strings.Contains(str, "ERROR_TYPE_BUSINESS") {
		t.Fatalf("expected error type in string: %q", str)
	}
	if !strings.Contains(str, "ERROR_CODE_FORBIDDEN") {
		t.Fatalf("expected error code in string: %q", str)
	}
	if !strings.Contains(str, "message") {
		t.Fatalf("expected message in string: %q", str)
	}
}
