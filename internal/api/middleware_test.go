package api

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCORS tests CORS middleware
func TestCORS(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		origin         string
		expectedStatus int
		checkHeaders   map[string]string
	}{
		{
			name:           "OPTIONS request returns 204",
			method:         "OPTIONS",
			origin:         "http://example.com",
			expectedStatus: http.StatusNoContent,
			checkHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
				"Access-Control-Allow-Headers": "Content-Type, Authorization",
			},
		},
		{
			name:           "GET request passes through",
			method:         "GET",
			origin:         "http://example.com",
			expectedStatus: http.StatusOK,
			checkHeaders: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := CORS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			}))

			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			for key, expected := range tt.checkHeaders {
				assert.Equal(t, expected, rec.Header().Get(key), "Header %s mismatch", key)
			}
		})
	}
}

// TestLogging tests logging middleware
func TestLogging(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := Logging()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/test?foo=bar", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "GET")
	assert.Contains(t, logOutput, "/test")
	assert.Contains(t, logOutput, "200")
}

// TestRecovery tests panic recovery middleware
func TestRecovery(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	tests := []struct {
		name          string
		handler       http.HandlerFunc
		expectedCode  int
		shouldPanic   bool
		panicMessage  string
	}{
		{
			name: "normal request",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			},
			expectedCode: http.StatusOK,
			shouldPanic:  false,
		},
		{
			name: "panic with string",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic("test panic")
			},
			expectedCode:  http.StatusInternalServerError,
			shouldPanic:   true,
			panicMessage:  "test panic",
		},
		{
			name: "panic with error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic(http.ErrAbortHandler)
			},
			expectedCode:  http.StatusInternalServerError,
			shouldPanic:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			handler := Recovery()(tt.handler)
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)

			if tt.shouldPanic {
				logOutput := buf.String()
				assert.Contains(t, logOutput, "panic")
				if tt.panicMessage != "" {
					assert.Contains(t, logOutput, tt.panicMessage)
				}
			}
		})
	}
}

// TestRequestID tests request ID middleware
func TestRequestID(t *testing.T) {
	tests := []struct {
		name           string
		existingID     string
		shouldGenerate bool
	}{
		{
			name:           "generates new request ID",
			existingID:     "",
			shouldGenerate: true,
		},
		{
			name:           "preserves existing request ID",
			existingID:     "existing-id-123",
			shouldGenerate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedID string
			handler := RequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedID = r.Header.Get("X-Request-ID")
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.existingID != "" {
				req.Header.Set("X-Request-ID", tt.existingID)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)

			if tt.shouldGenerate {
				require.NotEmpty(t, capturedID, "Request ID should be generated")
				assert.NotEqual(t, "", capturedID)
				assert.True(t, len(capturedID) > 10, "Generated ID should be reasonably long")
			} else {
				assert.Equal(t, tt.existingID, capturedID, "Existing ID should be preserved")
			}

			assert.Equal(t, capturedID, rec.Header().Get("X-Request-ID"))
		})
	}
}

// TestMiddlewareChaining tests that multiple middlewares work together
func TestMiddlewareChaining(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := Recovery()(
		Logging()(
			RequestID()(
				CORS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("OK"))
				})),
			),
		),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))
	assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
}
