package apierr

import (
	"encoding/json"
	"errors"
	"net/http"
)

func WriteError(w http.ResponseWriter, err error) {
  var appErr *AppError
  if errors.As(err, &appErr) {
    respond(w, statusFromType(appErr.Type), appErr.Type, appErr.Message)
    return
  }
  respond(w, http.StatusInternalServerError, TypeInternal, "internal server error")
}

func respond(w http.ResponseWriter, status int, code ErrorType, msg string) {
  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(status)
  json.NewEncoder(w).Encode(map[string]string{
    "error": msg,
    "code":  string(code),
  })
}

func statusFromType(t ErrorType) int {
  switch t {
		case TypeValidation:
			return http.StatusBadRequest
		case TypeNotFound:
			return http.StatusNotFound
		default:
			return http.StatusInternalServerError
  }
}