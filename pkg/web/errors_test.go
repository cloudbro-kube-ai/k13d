package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewAPIError(t *testing.T) {
	tests := []struct {
		code       string
		detail     string
		wantStatus int
		wantMsg    string
	}{
		{
			code:       ErrCodeBadRequest,
			detail:     "invalid input",
			wantStatus: http.StatusBadRequest,
		},
		{
			code:       ErrCodeUnauthorized,
			detail:     "missing token",
			wantStatus: http.StatusUnauthorized,
			wantMsg:    "Authentication required",
		},
		{
			code:       ErrCodeForbidden,
			detail:     "access denied",
			wantStatus: http.StatusForbidden,
			wantMsg:    "Access denied",
		},
		{
			code:       ErrCodeNotFound,
			detail:     "pod not found",
			wantStatus: http.StatusNotFound,
			wantMsg:    "Resource not found",
		},
		{
			code:       ErrCodeConflict,
			detail:     "already exists",
			wantStatus: http.StatusConflict,
		},
		{
			code:       ErrCodeValidation,
			detail:     "invalid field",
			wantStatus: http.StatusBadRequest,
		},
		{
			code:       ErrCodeK8sError,
			detail:     "cluster error",
			wantStatus: http.StatusBadGateway,
			wantMsg:    "Kubernetes API error",
		},
		{
			code:       ErrCodeLLMError,
			detail:     "ai error",
			wantStatus: http.StatusBadGateway,
		},
		{
			code:       ErrCodeLLMNotConfigured,
			detail:     "no config",
			wantStatus: http.StatusServiceUnavailable,
			wantMsg:    "AI assistant is not configured",
		},
		{
			code:       ErrCodeLLMNoToolCalling,
			detail:     "no tools",
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			code:       ErrCodeHelmError,
			detail:     "helm failed",
			wantStatus: http.StatusBadGateway,
		},
		{
			code:       ErrCodeDatabaseError,
			detail:     "db failed",
			wantStatus: http.StatusBadGateway,
		},
		{
			code:       ErrCodeTimeout,
			detail:     "request timeout",
			wantStatus: http.StatusGatewayTimeout,
			wantMsg:    "Request timed out",
		},
		{
			code:       ErrCodeRateLimited,
			detail:     "too many requests",
			wantStatus: http.StatusTooManyRequests,
			wantMsg:    "Rate limit exceeded",
		},
		{
			code:       ErrCodeInternalError,
			detail:     "something went wrong",
			wantStatus: http.StatusInternalServerError,
			wantMsg:    "An internal error occurred",
		},
		{
			code:       "UNKNOWN_CODE",
			detail:     "unknown error",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			err := NewAPIError(tt.code, tt.detail)

			if err.StatusCode != tt.wantStatus {
				t.Errorf("StatusCode = %d, want %d", err.StatusCode, tt.wantStatus)
			}

			if err.Detail != tt.detail {
				t.Errorf("Detail = %s, want %s", err.Detail, tt.detail)
			}

			if tt.wantMsg != "" && err.Message != tt.wantMsg {
				t.Errorf("Message = %s, want %s", err.Message, tt.wantMsg)
			}

			if err.Suggestion == "" {
				t.Error("Suggestion should not be empty")
			}
		})
	}
}

func TestNewAPIErrorWithSuggestion(t *testing.T) {
	customSuggestion := "Custom suggestion here"
	err := NewAPIErrorWithSuggestion(ErrCodeK8sError, "some error", customSuggestion)

	if err.Suggestion != customSuggestion {
		t.Errorf("Suggestion = %s, want %s", err.Suggestion, customSuggestion)
	}

	// Empty suggestion should keep default
	err2 := NewAPIErrorWithSuggestion(ErrCodeK8sError, "some error", "")
	if err2.Suggestion == "" {
		t.Error("Suggestion should not be empty when custom suggestion is empty")
	}
}

func TestWriteError(t *testing.T) {
	rr := httptest.NewRecorder()
	err := NewAPIError(ErrCodeNotFound, "pod not found")

	WriteError(rr, err)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusNotFound)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", contentType)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "NOT_FOUND") {
		t.Errorf("Body should contain error code, got: %s", body)
	}
}

func TestWriteErrorSimple(t *testing.T) {
	tests := []struct {
		status   int
		message  string
		wantCode string
	}{
		{http.StatusBadRequest, "bad input", ErrCodeBadRequest},
		{http.StatusUnauthorized, "not logged in", ErrCodeUnauthorized},
		{http.StatusForbidden, "no access", ErrCodeForbidden},
		{http.StatusNotFound, "missing", ErrCodeNotFound},
		{http.StatusInternalServerError, "oops", ErrCodeInternalError},
	}

	for _, tt := range tests {
		t.Run(tt.wantCode, func(t *testing.T) {
			rr := httptest.NewRecorder()
			WriteErrorSimple(rr, tt.status, tt.message)

			if rr.Code != tt.status {
				t.Errorf("Status = %d, want %d", rr.Code, tt.status)
			}

			body := rr.Body.String()
			if !strings.Contains(body, tt.wantCode) {
				t.Errorf("Body should contain %s, got: %s", tt.wantCode, body)
			}
		})
	}
}

func TestParseK8sError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode string
	}{
		{
			name:     "nil error",
			err:      nil,
			wantCode: "",
		},
		{
			name:     "not found error",
			err:      errors.New("pod nginx not found"),
			wantCode: ErrCodeNotFound,
		},
		{
			name:     "forbidden error",
			err:      errors.New("forbidden: cannot access namespace"),
			wantCode: ErrCodeForbidden,
		},
		{
			name:     "unauthorized error",
			err:      errors.New("unauthorized: token expired"),
			wantCode: ErrCodeUnauthorized,
		},
		{
			name:     "connection refused",
			err:      errors.New("connection refused to api server"),
			wantCode: ErrCodeK8sError,
		},
		{
			name:     "timeout error",
			err:      errors.New("context deadline exceeded (timeout)"),
			wantCode: ErrCodeTimeout,
		},
		{
			name:     "already exists",
			err:      errors.New("deployment nginx already exists"),
			wantCode: ErrCodeConflict,
		},
		{
			name:     "generic error",
			err:      errors.New("some random error"),
			wantCode: ErrCodeK8sError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseK8sError(tt.err)

			if tt.wantCode == "" {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("Expected non-nil result")
			}

			if result.Code != tt.wantCode {
				t.Errorf("Code = %s, want %s", result.Code, tt.wantCode)
			}
		})
	}
}

func TestParseLLMError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		provider string
		wantCode string
	}{
		{
			name:     "nil error",
			err:      nil,
			provider: "OpenAI",
			wantCode: "",
		},
		{
			name:     "401 unauthorized",
			err:      errors.New("API returned 401: unauthorized"),
			provider: "OpenAI",
			wantCode: ErrCodeUnauthorized,
		},
		{
			name:     "invalid api key",
			err:      errors.New("invalid api key provided"),
			provider: "Anthropic",
			wantCode: ErrCodeUnauthorized,
		},
		{
			name:     "rate limit 429",
			err:      errors.New("API returned 429: rate limit exceeded"),
			provider: "OpenAI",
			wantCode: ErrCodeRateLimited,
		},
		{
			name:     "connection refused",
			err:      errors.New("dial tcp: connection refused"),
			provider: "Ollama",
			wantCode: ErrCodeLLMError,
		},
		{
			name:     "no such host",
			err:      errors.New("dial tcp: no such host"),
			provider: "Custom",
			wantCode: ErrCodeLLMError,
		},
		{
			name:     "timeout",
			err:      errors.New("request timeout after 30s"),
			provider: "OpenAI",
			wantCode: ErrCodeTimeout,
		},
		{
			name:     "model not found",
			err:      errors.New("model gpt-5 not found"),
			provider: "OpenAI",
			wantCode: ErrCodeLLMError,
		},
		{
			name:     "generic error",
			err:      errors.New("unexpected error"),
			provider: "OpenAI",
			wantCode: ErrCodeLLMError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseLLMError(tt.err, tt.provider)

			if tt.wantCode == "" {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("Expected non-nil result")
			}

			if result.Code != tt.wantCode {
				t.Errorf("Code = %s, want %s", result.Code, tt.wantCode)
			}

			// Check that provider is mentioned in suggestion for connection-related errors
			// Note: Not all LLM errors include the provider in the suggestion
			if (tt.name == "connection refused" || tt.name == "no such host") &&
				!strings.Contains(result.Suggestion, tt.provider) {
				t.Errorf("Suggestion should mention provider %s for connection errors, got: %s", tt.provider, result.Suggestion)
			}
		})
	}
}

func TestFriendlyErrorMessage(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantContain string
	}{
		{
			name:        "nil error",
			err:         nil,
			wantContain: "",
		},
		{
			name:        "connection refused",
			err:         errors.New("dial tcp: connection refused"),
			wantContain: "Unable to connect",
		},
		{
			name:        "no such host",
			err:         errors.New("no such host: example.com"),
			wantContain: "server address could not be found",
		},
		{
			name:        "timeout",
			err:         errors.New("context deadline exceeded (timeout)"),
			wantContain: "took too long",
		},
		{
			name:        "permission denied",
			err:         errors.New("permission denied for user"),
			wantContain: "don't have permission",
		},
		{
			name:        "not found",
			err:         errors.New("resource not found"),
			wantContain: "was not found",
		},
		{
			name:        "already exists",
			err:         errors.New("resource already exists"),
			wantContain: "already exists",
		},
		{
			name:        "invalid input",
			err:         errors.New("invalid field value"),
			wantContain: "invalid",
		},
		{
			name:        "generic error",
			err:         errors.New("something went wrong"),
			wantContain: "An error occurred",
		},
		{
			name:        "long error truncated",
			err:         errors.New(strings.Repeat("a", 300)),
			wantContain: "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FriendlyErrorMessage(tt.err)

			if tt.wantContain == "" {
				if result != "" {
					t.Errorf("Expected empty string, got: %s", result)
				}
				return
			}

			if !strings.Contains(result, tt.wantContain) {
				t.Errorf("Result should contain %q, got: %s", tt.wantContain, result)
			}
		})
	}
}

func TestGetStatusCodeForError(t *testing.T) {
	// Test all error codes have appropriate status codes
	tests := []struct {
		code       string
		wantStatus int
	}{
		{ErrCodeBadRequest, http.StatusBadRequest},
		{ErrCodeValidation, http.StatusBadRequest},
		{ErrCodeUnauthorized, http.StatusUnauthorized},
		{ErrCodeForbidden, http.StatusForbidden},
		{ErrCodeNotFound, http.StatusNotFound},
		{ErrCodeConflict, http.StatusConflict},
		{ErrCodeTimeout, http.StatusGatewayTimeout},
		{ErrCodeRateLimited, http.StatusTooManyRequests},
		{ErrCodeLLMNotConfigured, http.StatusServiceUnavailable},
		{ErrCodeLLMNoToolCalling, http.StatusServiceUnavailable},
		{ErrCodeK8sError, http.StatusBadGateway},
		{ErrCodeHelmError, http.StatusBadGateway},
		{ErrCodeDatabaseError, http.StatusBadGateway},
		{ErrCodeLLMError, http.StatusBadGateway},
		{ErrCodeInternalError, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			status := getStatusCodeForError(tt.code)
			if status != tt.wantStatus {
				t.Errorf("getStatusCodeForError(%s) = %d, want %d", tt.code, status, tt.wantStatus)
			}
		})
	}
}
