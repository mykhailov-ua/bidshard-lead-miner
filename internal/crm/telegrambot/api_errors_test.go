package telegrambot

import (
	"strings"
	"testing"
)

func TestAPIErrorStringAndHint(t *testing.T) {
	t.Parallel()
	e := &APIError{
		Method:      "sendMessage",
		HTTPStatus:  403,
		ErrorCode:   403,
		Description: "Forbidden: bot was blocked by the user",
	}
	if !strings.Contains(e.String(), "code=403") {
		t.Fatalf("string=%q", e.String())
	}
	if e.Hint() == "" {
		t.Fatal("expected hint")
	}
}

func TestDecodeEnvelopeNotOK(t *testing.T) {
	t.Parallel()
	env, err := decodeEnvelope([]byte(`{"ok":false,"error_code":401,"description":"Unauthorized"}`))
	if err != nil {
		t.Fatal(err)
	}
	ae := apiErrorFromEnvelope("getMe", 200, env)
	if ae == nil || ae.ErrorCode != 401 {
		t.Fatalf("api error=%v", ae)
	}
}
