package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
)

const (
	CodeValidation   = "VALIDATION_ERROR"
	CodeDomain       = "DOMAIN_ERROR"
	CodeUnauthorized = "UNAUTHORIZED"
	CodeForbidden    = "FORBIDDEN"
	CodeNotFound     = "NOT_FOUND"
	CodeRateLimited  = "RATE_LIMITED"
	CodeUnavailable  = "SERVICE_UNAVAILABLE"
	CodeInternal     = "INTERNAL_ERROR"
)

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
	Details []string          `json:"details,omitempty"`
}

type APIError struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
	Details []string
}

func (e *APIError) Error() string {
	return e.Code
}

func ValidationError(fields map[string]string) *APIError {
	return &APIError{
		Status:  http.StatusBadRequest,
		Code:    CodeValidation,
		Message: "Некоторые поля заполнены неверно",
		Fields:  fields,
	}
}

func DomainError(code string, message string) *APIError {
	if code == "" {
		code = CodeDomain
	}
	if message == "" {
		message = "Операция не может быть выполнена"
	}

	return &APIError{
		Status:  http.StatusConflict,
		Code:    code,
		Message: message,
	}
}

func UnauthorizedError() *APIError {
	return &APIError{
		Status:  http.StatusUnauthorized,
		Code:    CodeUnauthorized,
		Message: "Требуется авторизация",
	}
}

func ForbiddenError() *APIError {
	return &APIError{
		Status:  http.StatusForbidden,
		Code:    CodeForbidden,
		Message: "Недостаточно прав",
	}
}

func NotFoundError() *APIError {
	return &APIError{
		Status:  http.StatusNotFound,
		Code:    CodeNotFound,
		Message: "Объект не найден",
	}
}

func RateLimitedError() *APIError {
	return &APIError{
		Status:  http.StatusTooManyRequests,
		Code:    CodeRateLimited,
		Message: "Слишком много попыток. Попробуйте позже",
	}
}

func ServiceUnavailableError() *APIError {
	return &APIError{
		Status:  http.StatusServiceUnavailable,
		Code:    CodeUnavailable,
		Message: "Сервис временно недоступен",
	}
}

func InternalError() *APIError {
	return &APIError{
		Status:  http.StatusInternalServerError,
		Code:    CodeInternal,
		Message: "Внутренняя ошибка сервера",
	}
}

func RespondError(w http.ResponseWriter, err error) {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		apiErr = InternalError()
	}

	WriteJSON(w, apiErr.Status, ErrorResponse{
		Error: ErrorBody{
			Code:    apiErr.Code,
			Message: apiErr.Message,
			Fields:  apiErr.Fields,
			Details: apiErr.Details,
		},
	})
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
