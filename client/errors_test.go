package client

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func response(status int, header http.Header, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body))}
}

var jsonHeader = http.Header{"Content-Type": {"application/json; charset=utf-8"}}

func TestDecode(t *testing.T) {
	got, err := Decode[*UserDto](response(200, jsonHeader, `{"id":"u1"}`), nil)
	if err != nil || got.ID != "u1" {
		t.Fatalf("got %+v, %v", got, err)
	}

	// The frontend answers unknown paths with its HTML index page.
	_, err = Decode[*UserDto](response(200, http.Header{"Content-Type": {"text/html"}}, "<html>"), nil)
	if err == nil || AsAPIError(err) != nil {
		t.Fatalf("want a non-API error for a non-JSON 2xx, got %v", err)
	}

	sent := errors.New("connection refused")
	if _, err := Decode[*UserDto](nil, sent); err != sent {
		t.Fatalf("want the transport error, got %v", err)
	}
}

func TestCheck(t *testing.T) {
	if err := Check(response(204, nil, ""), nil); err != nil {
		t.Fatal(err)
	}
}

func TestAPIError(t *testing.T) {
	header := http.Header{"Content-Type": {"application/json"}, "Retry-After": {"30"}, "X-Request-Id": {"header-id"}}
	body := `{"error":"Request validation failed","code":"validation_failed","request_id":"body-id",
		"details":{"fields":[{"field":"logoUrl","code":"url","message":"must be a URL"}]}}`
	err := fmt.Errorf("create client: %w", Check(response(400, header, body), nil))

	apiErr := AsAPIError(err)
	if apiErr == nil || apiErr.StatusCode != 400 || apiErr.Code != CodeValidationFailed ||
		apiErr.RequestID != "body-id" || apiErr.RetryAfter != 30*time.Second {
		t.Fatalf("got %+v", apiErr)
	}
	if !apiErr.HasFieldError("darkLogoUrl", "logoUrl") || apiErr.HasFieldError("name") {
		t.Fatal("HasFieldError doesn't match details.fields")
	}
	if want := "create client: pocket id: 400 Request validation failed (code validation_failed, request ID body-id)"; err.Error() != want {
		t.Fatalf("got %q, want %q", err, want)
	}
}

func TestAPIErrorWithoutErrorDto(t *testing.T) {
	for _, body := range []string{"<html>Bad Gateway</html>", `{"code":502}`} {
		header := http.Header{"X-Request-Id": {"header-id"}}
		apiErr := AsAPIError(Check(response(502, header, body), nil))
		if apiErr == nil || apiErr.StatusCode != 502 || apiErr.Code != "" || apiErr.RequestID != "header-id" ||
			string(apiErr.Body) != body || apiErr.Error() != "pocket id: 502 Bad Gateway" {
			t.Fatalf("body %q: got %+v", body, apiErr)
		}
	}
}
