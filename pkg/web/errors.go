package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// APIError represents a user-friendly error response
type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Detail     string `json:"detail,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
	StatusCode int    `json:"-"`
}

// Error codes for categorization
const (
	ErrCodeInternalError    = "INTERNAL_ERROR"
	ErrCodeBadRequest       = "BAD_REQUEST"
	ErrCodeUnauthorized     = "UNAUTHORIZED"
	ErrCodeForbidden        = "FORBIDDEN"
	ErrCodeNotFound         = "NOT_FOUND"
	ErrCodeConflict         = "CONFLICT"
	ErrCodeValidation       = "VALIDATION_ERROR"
	ErrCodeK8sError         = "K8S_ERROR"
	ErrCodeLLMError         = "LLM_ERROR"
	ErrCodeLLMNotConfigured = "LLM_NOT_CONFIGURED"
	ErrCodeLLMNoToolCalling = "LLM_NO_TOOL_CALLING"
	ErrCodeHelmError        = "HELM_ERROR"
	ErrCodeDatabaseError    = "DATABASE_ERROR"
	ErrCodeTimeout          = "TIMEOUT"
	ErrCodeRateLimited      = "RATE_LIMITED"
)

// Common error messages with user-friendly suggestions
var errorMessages = map[string]struct {
	Message    string
	Suggestion string
}{
	ErrCodeInternalError: {
		Message:    "An internal error occurred",
		Suggestion: "Please try again. If the problem persists, check the server logs or contact support.",
	},
	ErrCodeUnauthorized: {
		Message:    "Authentication required",
		Suggestion: "Please log in to access this resource.",
	},
	ErrCodeForbidden: {
		Message:    "Access denied",
		Suggestion: "You don't have permission to perform this action. Contact your administrator for access.",
	},
	ErrCodeNotFound: {
		Message:    "Resource not found",
		Suggestion: "The requested resource doesn't exist or may have been deleted.",
	},
	ErrCodeLLMNotConfigured: {
		Message:    "AI assistant is not configured",
		Suggestion: "Go to Settings > AI/LLM Settings to configure your LLM provider and API key.",
	},
	ErrCodeLLMNoToolCalling: {
		Message:    "The selected LLM model doesn't support tool calling",
		Suggestion: "Enable JSON mode in Settings, or switch to a model that supports function calling (e.g., GPT-4, Claude 3).",
	},
	ErrCodeK8sError: {
		Message:    "Kubernetes API error",
		Suggestion: "Check your cluster connection and permissions. Ensure your kubeconfig is valid.",
	},
	ErrCodeHelmError: {
		Message:    "Helm operation failed",
		Suggestion: "Check your Helm configuration and ensure the chart/release exists.",
	},
	ErrCodeTimeout: {
		Message:    "Request timed out",
		Suggestion: "The operation took too long. Try again with a smaller scope or check your network connection.",
	},
	ErrCodeRateLimited: {
		Message:    "Rate limit exceeded",
		Suggestion: "You've made too many requests. Please wait a moment before trying again.",
	},
}

// NewAPIError creates a new API error with a user-friendly message
func NewAPIError(code string, detail string) *APIError {
	info := errorMessages[code]
	if info.Message == "" {
		info = errorMessages[ErrCodeInternalError]
	}

	statusCode := getStatusCodeForError(code)

	return &APIError{
		Code:       code,
		Message:    info.Message,
		Detail:     detail,
		Suggestion: info.Suggestion,
		StatusCode: statusCode,
	}
}

// NewAPIErrorWithSuggestion creates a new API error with a custom suggestion
func NewAPIErrorWithSuggestion(code, detail, suggestion string) *APIError {
	err := NewAPIError(code, detail)
	if suggestion != "" {
		err.Suggestion = suggestion
	}
	return err
}

func getStatusCodeForError(code string) int {
	switch code {
	case ErrCodeBadRequest, ErrCodeValidation:
		return http.StatusBadRequest
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrCodeForbidden:
		return http.StatusForbidden
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeConflict:
		return http.StatusConflict
	case ErrCodeTimeout:
		return http.StatusGatewayTimeout
	case ErrCodeRateLimited:
		return http.StatusTooManyRequests
	case ErrCodeLLMNotConfigured, ErrCodeLLMNoToolCalling:
		return http.StatusServiceUnavailable
	case ErrCodeK8sError, ErrCodeHelmError, ErrCodeDatabaseError, ErrCodeLLMError:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// WriteError writes an API error to the response
func WriteError(w http.ResponseWriter, err *APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.StatusCode)
	_ = json.NewEncoder(w).Encode(err)
}

// WriteErrorSimple writes a simple error message (backward compatibility)
func WriteErrorSimple(w http.ResponseWriter, statusCode int, message string) {
	code := ErrCodeInternalError
	switch statusCode {
	case http.StatusBadRequest:
		code = ErrCodeBadRequest
	case http.StatusUnauthorized:
		code = ErrCodeUnauthorized
	case http.StatusForbidden:
		code = ErrCodeForbidden
	case http.StatusNotFound:
		code = ErrCodeNotFound
	}

	err := NewAPIError(code, message)
	err.StatusCode = statusCode
	WriteError(w, err)
}

// ParseK8sError converts Kubernetes errors to user-friendly messages
func ParseK8sError(err error) *APIError {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Check for common Kubernetes error patterns
	switch {
	case strings.Contains(errStr, "not found"):
		return NewAPIError(ErrCodeNotFound, errStr)
	case strings.Contains(errStr, "forbidden"):
		return NewAPIErrorWithSuggestion(ErrCodeForbidden, errStr,
			"Check your RBAC permissions. The service account may need additional roles.")
	case strings.Contains(errStr, "unauthorized"):
		return NewAPIErrorWithSuggestion(ErrCodeUnauthorized, errStr,
			"Your authentication token may have expired. Try refreshing your kubeconfig.")
	case strings.Contains(errStr, "connection refused"):
		return NewAPIErrorWithSuggestion(ErrCodeK8sError, errStr,
			"Cannot connect to the Kubernetes API server. Check if the cluster is running and accessible.")
	case strings.Contains(errStr, "timeout"):
		return NewAPIError(ErrCodeTimeout, errStr)
	case strings.Contains(errStr, "already exists"):
		return NewAPIError(ErrCodeConflict, errStr)
	default:
		return NewAPIError(ErrCodeK8sError, errStr)
	}
}

// ParseLLMError converts LLM errors to user-friendly messages
func ParseLLMError(err error, provider string) *APIError {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	switch {
	case strings.Contains(errStr, "401") || strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "invalid api key"):
		return NewAPIErrorWithSuggestion(ErrCodeUnauthorized, errStr,
			fmt.Sprintf("Your %s API key appears to be invalid. Check Settings > AI/LLM Settings.", provider))
	case strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit"):
		return NewAPIErrorWithSuggestion(ErrCodeRateLimited, errStr,
			"You've exceeded the API rate limit. Wait a moment or upgrade your API plan.")
	case strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "no such host"):
		return NewAPIErrorWithSuggestion(ErrCodeLLMError, errStr,
			fmt.Sprintf("Cannot connect to %s. Check your endpoint URL and network connection.", provider))
	case strings.Contains(errStr, "timeout"):
		return NewAPIErrorWithSuggestion(ErrCodeTimeout, errStr,
			"The LLM request timed out. Try a simpler question or check your network.")
	case strings.Contains(errStr, "model") && strings.Contains(errStr, "not found"):
		return NewAPIErrorWithSuggestion(ErrCodeLLMError, errStr,
			"The specified model was not found. Check if the model name is correct in Settings.")
	default:
		return NewAPIError(ErrCodeLLMError, errStr)
	}
}

// HTTPError is a convenience wrapper for common HTTP error responses.
// Use this as a drop-in replacement for http.Error() calls.
func HTTPError(w http.ResponseWriter, message string, statusCode int) {
	WriteErrorSimple(w, statusCode, message)
}

// BadRequest writes a 400 Bad Request error response
func BadRequest(w http.ResponseWriter, message string) {
	WriteError(w, NewAPIError(ErrCodeBadRequest, message))
}

// Unauthorized writes a 401 Unauthorized error response
func Unauthorized(w http.ResponseWriter, message string) {
	WriteError(w, NewAPIError(ErrCodeUnauthorized, message))
}

// Forbidden writes a 403 Forbidden error response
func Forbidden(w http.ResponseWriter, message string) {
	WriteError(w, NewAPIError(ErrCodeForbidden, message))
}

// NotFound writes a 404 Not Found error response
func NotFound(w http.ResponseWriter, message string) {
	WriteError(w, NewAPIError(ErrCodeNotFound, message))
}

// InternalError writes a 500 Internal Server Error response
func InternalError(w http.ResponseWriter, message string) {
	WriteError(w, NewAPIError(ErrCodeInternalError, message))
}

// K8sError writes an error response for Kubernetes API errors
func K8sError(w http.ResponseWriter, err error) {
	WriteError(w, ParseK8sError(err))
}

// LLMError writes an error response for LLM API errors
func LLMError(w http.ResponseWriter, err error, provider string) {
	WriteError(w, ParseLLMError(err, provider))
}

// MethodNotAllowed writes a 405 Method Not Allowed response
func MethodNotAllowed(w http.ResponseWriter, allowedMethods ...string) {
	if len(allowedMethods) > 0 {
		w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
	}
	WriteErrorSimple(w, http.StatusMethodNotAllowed, "Method not allowed")
}

// FriendlyErrorMessage returns a user-friendly error message for display
func FriendlyErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	errStr := err.Error()

	// Map common technical errors to friendly messages
	friendlyMessages := map[string]string{
		"connection refused": "Unable to connect to the server. Please check if the service is running.",
		"no such host":       "The server address could not be found. Please check the URL.",
		"timeout":            "The request took too long. Please try again.",
		"permission denied":  "You don't have permission to perform this action.",
		"not found":          "The requested resource was not found.",
		"already exists":     "This resource already exists.",
		"invalid":            "The provided data is invalid. Please check your input.",
	}

	for pattern, friendly := range friendlyMessages {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return friendly
		}
	}

	// If no pattern matches, return a generic message with the original error
	if len(errStr) > 200 {
		errStr = errStr[:200] + "..."
	}
	return fmt.Sprintf("An error occurred: %s", errStr)
}
