package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// ErrorLevel represents the severity of an error
type ErrorLevel string

const (
	ErrorLevelInfo    ErrorLevel = "info"
	ErrorLevelWarning ErrorLevel = "warning"
	ErrorLevelError   ErrorLevel = "error"
	ErrorLevelCritical ErrorLevel = "critical"
)

// AppError represents a structured application error
type AppError struct {
	Code       int                    `json:"code"`
	Message    string                 `json:"message"`
	Details    string                 `json:"details,omitempty"`
	Level      ErrorLevel             `json:"level"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	RequestID  string                 `json:"request_id,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	Stacktrace string                 `json:"stacktrace,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Details)
}

// ErrorHandler provides centralized error handling
type ErrorHandler struct {
	includeStackTrace bool
	logLevel          ErrorLevel
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(includeStackTrace bool, logLevel ErrorLevel) *ErrorHandler {
	return &ErrorHandler{
		includeStackTrace: includeStackTrace,
		logLevel:          logLevel,
	}
}

// HandleError processes and logs application errors
func (eh *ErrorHandler) HandleError(w http.ResponseWriter, r *http.Request, err error, code int, userMessage string) {
	appErr := eh.createAppError(r, err, code, userMessage)
	eh.logError(appErr)
	eh.writeErrorResponse(w, r, appErr)
}

// HandleHTTPError handles standard HTTP errors with proper logging
func (eh *ErrorHandler) HandleHTTPError(w http.ResponseWriter, r *http.Request, code int, message string, details ...string) {
	var detail string
	if len(details) > 0 {
		detail = strings.Join(details, "; ")
	}
	
	appErr := eh.createAppError(r, fmt.Errorf(message), code, message)
	if detail != "" {
		appErr.Details = detail
	}
	
	eh.logError(appErr)
	eh.writeErrorResponse(w, r, appErr)
}

// createAppError creates a structured AppError
func (eh *ErrorHandler) createAppError(r *http.Request, err error, code int, userMessage string) *AppError {
	appErr := &AppError{
		Code:      code,
		Message:   userMessage,
		Details:   err.Error(),
		Level:     eh.getErrorLevel(code),
		Context:   eh.extractContext(r),
		Timestamp: time.Now(),
		RequestID: middleware.GetReqID(r.Context()),
	}

	// Extract user ID if available
	if account := r.Context().Value("account"); account != nil {
		if acc, ok := account.(interface{ GetID() int }); ok {
			appErr.UserID = fmt.Sprintf("%d", acc.GetID())
		}
	}

	// Add stack trace for 5xx errors or if explicitly enabled
	if eh.includeStackTrace && (code >= 500 || eh.logLevel == ErrorLevelError) {
		appErr.Stacktrace = eh.getStackTrace()
	}

	return appErr
}

// getErrorLevel determines the error level based on HTTP status code
func (eh *ErrorHandler) getErrorLevel(code int) ErrorLevel {
	switch {
	case code >= 500:
		return ErrorLevelError
	case code >= 400:
		return ErrorLevelWarning
	case code >= 300:
		return ErrorLevelInfo
	default:
		return ErrorLevelInfo
	}
}

// extractContext extracts relevant context from the request
func (eh *ErrorHandler) extractContext(r *http.Request) map[string]interface{} {
	context := map[string]interface{}{
		"method":     r.Method,
		"path":       r.URL.Path,
		"query":      r.URL.RawQuery,
		"user_agent": r.UserAgent(),
		"remote_ip":  r.RemoteAddr,
	}

	// Add referer if present
	if referer := r.Header.Get("Referer"); referer != "" {
		context["referer"] = referer
	}

	// Add content type if present
	if contentType := r.Header.Get("Content-Type"); contentType != "" {
		context["content_type"] = contentType
	}

	// Add form data for POST requests (excluding sensitive fields)
	if r.Method == "POST" && strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err == nil {
			formData := make(map[string]string)
			for key, values := range r.Form {
				// Skip sensitive fields
				if strings.Contains(strings.ToLower(key), "password") ||
					strings.Contains(strings.ToLower(key), "token") ||
					strings.Contains(strings.ToLower(key), "secret") {
					formData[key] = "[REDACTED]"
				} else if len(values) > 0 {
					formData[key] = values[0]
				}
			}
			context["form_data"] = formData
		}
	}

	return context
}

// getStackTrace returns the current stack trace
func (eh *ErrorHandler) getStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// logError logs the error with appropriate level
func (eh *ErrorHandler) logError(appErr *AppError) {
	// Convert to JSON for structured logging
	jsonBytes, err := json.Marshal(appErr)
	if err != nil {
		log.Printf("ERROR: Failed to marshal error to JSON: %v", err)
		log.Printf("ORIGINAL ERROR: [%d] %s: %s", appErr.Code, appErr.Message, appErr.Details)
		return
	}

	// Log based on error level
	switch appErr.Level {
	case ErrorLevelCritical:
		log.Printf("CRITICAL: %s", string(jsonBytes))
	case ErrorLevelError:
		log.Printf("ERROR: %s", string(jsonBytes))
	case ErrorLevelWarning:
		log.Printf("WARNING: %s", string(jsonBytes))
	case ErrorLevelInfo:
		log.Printf("INFO: %s", string(jsonBytes))
	default:
		log.Printf("LOG: %s", string(jsonBytes))
	}
}

// writeErrorResponse writes the error response to the client
func (eh *ErrorHandler) writeErrorResponse(w http.ResponseWriter, r *http.Request, appErr *AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.Code)

	// Create client-safe response (don't expose internal details)
	clientResponse := map[string]interface{}{
		"error":      true,
		"code":       appErr.Code,
		"message":    appErr.Message,
		"timestamp":  appErr.Timestamp.Format(time.RFC3339),
		"request_id": appErr.RequestID,
	}

	// Only include details for 4xx errors or in development
	if appErr.Code < 500 {
		clientResponse["details"] = appErr.Details
	}

	if err := json.NewEncoder(w).Encode(clientResponse); err != nil {
		log.Printf("ERROR: Failed to write error response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ErrorMiddleware creates middleware for automatic error handling
func (eh *ErrorHandler) ErrorMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap the ResponseWriter to capture status codes
			wrapped := &responseWrapper{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				errorHandler:   eh,
				request:        r,
			}

			// Add error handler to context for use in handlers
			ctx := context.WithValue(r.Context(), "error_handler", eh)
			
			// Call the next handler
			next.ServeHTTP(wrapped, r.WithContext(ctx))
		})
	}
}

// responseWrapper wraps http.ResponseWriter to capture status codes
type responseWrapper struct {
	http.ResponseWriter
	statusCode   int
	errorHandler *ErrorHandler
	request      *http.Request
	written      bool
}

func (rw *responseWrapper) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.written = true

	// Log 4xx and 5xx responses
	if statusCode >= 400 {
		message := http.StatusText(statusCode)
		rw.errorHandler.logError(rw.errorHandler.createAppError(
			rw.request,
			fmt.Errorf("HTTP %d: %s", statusCode, message),
			statusCode,
			message,
		))
	}

	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWrapper) Write(data []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(data)
}

// Helper functions for handlers to use

// GetErrorHandler extracts the error handler from context
func GetErrorHandler(ctx context.Context) *ErrorHandler {
	if eh, ok := ctx.Value("error_handler").(*ErrorHandler); ok {
		return eh
	}
	// Fallback to default error handler
	return NewErrorHandler(false, ErrorLevelWarning)
}

// HTTPError is a convenience function for handlers
func HTTPError(w http.ResponseWriter, r *http.Request, code int, message string, err error) {
	eh := GetErrorHandler(r.Context())
	if err != nil {
		eh.HandleError(w, r, err, code, message)
	} else {
		eh.HandleHTTPError(w, r, code, message)
	}
}

// HTTPErrorf is like HTTPError but with formatted message
func HTTPErrorf(w http.ResponseWriter, r *http.Request, code int, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	HTTPError(w, r, code, message, nil)
}