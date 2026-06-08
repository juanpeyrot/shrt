package apierr

type ErrorType string

const (
  TypeValidation ErrorType = "VALIDATION"
  TypeNotFound   ErrorType = "NOT_FOUND"
  TypeInternal   ErrorType = "INTERNAL"
)

type AppError struct {
  Type    ErrorType
  Message string
  Err     error
}

func (e *AppError) Error() string  { return e.Message }
func (e *AppError) Unwrap() error  { return e.Err }