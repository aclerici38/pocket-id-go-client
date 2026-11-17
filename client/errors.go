package client

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"slices"
	"strconv"
	"time"
)

// Code is the stable error code of an APIError. Its constants are generated from upstream's apperror package.
type Code string

// APIError is a non-2xx response. A body that isn't Pocket ID's ErrorDto, such as a proxy's
// error page, leaves Code, Message and Details empty.
type APIError struct {
	StatusCode int
	Code       Code
	Message    string
	RequestID  string
	// Details is extra context such as {"property": "email"}. Validation errors list the
	// fields they reject under "fields".
	Details map[string]any
	// RetryAfter is how long a rate-limited caller should wait.
	RetryAfter time.Duration
	Body       []byte
}

func (e *APIError) Error() string {
	s := fmt.Sprintf("pocket id: %d %s", e.StatusCode, cmp.Or(e.Message, http.StatusText(e.StatusCode)))
	if e.Code != "" {
		s += fmt.Sprintf(" (code %s, request ID %s)", e.Code, e.RequestID)
	}
	return s
}

// HasFieldError reports whether a validation error rejects any of fields.
func (e *APIError) HasFieldError(fields ...string) bool {
	entries, _ := e.Details["fields"].([]any)
	return slices.ContainsFunc(entries, func(entry any) bool {
		field, _ := entry.(map[string]any)["field"].(string)
		return slices.Contains(fields, field)
	})
}

// AsAPIError returns the *APIError in err's chain, or nil if there is none.
func AsAPIError(err error) *APIError {
	apiErr, _ := errors.AsType[*APIError](err)
	return apiErr
}

// Decode returns the JSON body of a 2xx response, as in client.Decode[*client.UserDto](c.GetUserByID(ctx, id)).
func Decode[T any](rsp *http.Response, err error) (T, error) {
	body, err := read(rsp, err)
	if err != nil {
		return *new(T), err
	}
	ct := rsp.Header.Get("Content-Type")
	if t, _, _ := mime.ParseMediaType(ct); t != "application/json" {
		return *new(T), fmt.Errorf("pocket id: %d response is %q, not JSON", rsp.StatusCode, ct)
	}
	var v T
	if err := json.Unmarshal(body, &v); err != nil {
		return *new(T), fmt.Errorf("pocket id: decode response: %w", err)
	}
	return v, nil
}

// Check discards the body of a 2xx response, as in client.Check(c.DeleteUser(ctx, id)).
func Check(rsp *http.Response, err error) error {
	_, err = read(rsp, err)
	return err
}

// read returns the body of a 2xx response and turns any other response into an *APIError.
func read(rsp *http.Response, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, fmt.Errorf("pocket id: read response: %w", err)
	}
	if rsp.StatusCode >= 200 && rsp.StatusCode < 300 {
		return body, nil
	}

	var dto struct {
		Code      Code           `json:"code"`
		Error     string         `json:"error"`
		RequestID string         `json:"request_id"`
		Details   map[string]any `json:"details"`
	}
	_ = json.Unmarshal(body, &dto)
	// Pocket ID sends whole seconds, never the HTTP-date form.
	retryAfter, _ := strconv.Atoi(rsp.Header.Get("Retry-After"))
	return nil, &APIError{
		StatusCode: rsp.StatusCode,
		Code:       dto.Code,
		Message:    dto.Error,
		RequestID:  cmp.Or(dto.RequestID, rsp.Header.Get("X-Request-ID")),
		Details:    dto.Details,
		RetryAfter: time.Duration(max(retryAfter, 0)) * time.Second,
		Body:       body,
	}
}
