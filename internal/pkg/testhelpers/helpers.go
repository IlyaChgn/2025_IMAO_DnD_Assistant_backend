package testhelpers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// NewTestContext returns a context suitable for tests.
// logger.FromContext will return the no-op logger for a plain context,
// so no special setup is needed.
func NewTestContext() context.Context {
	return context.Background()
}

// MustJSON marshals v to JSON bytes, failing the test on error.
func MustJSON(t *testing.T, v any) []byte {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("MustJSON: failed to marshal: %v", err)
	}

	return data
}

// DoRequest creates an httptest request, executes it via handler, and returns the recorder.
func DoRequest(t *testing.T, handler http.HandlerFunc, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()

	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	return rr
}

// DecodeJSON decodes the response body into dst, failing the test on error.
func DecodeJSON(t *testing.T, body *bytes.Buffer, dst any) {
	t.Helper()

	if err := json.NewDecoder(body).Decode(dst); err != nil {
		t.Fatalf("DecodeJSON: failed to decode: %v", err)
	}
}

// DecodeErrorResponse is a convenience wrapper that decodes the response body
// into models.ErrResponse and returns the Status field. It fails the test if
// decoding fails.
func DecodeErrorResponse(t *testing.T, body *bytes.Buffer) string {
	t.Helper()

	var resp models.ErrResponse
	DecodeJSON(t, body, &resp)

	return resp.Status
}
