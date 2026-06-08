package apierr

func NewValidation(msg string) *AppError {
    return &AppError{Type: TypeValidation, Message: msg}
}

func NewNotFound(msg string) *AppError {
    return &AppError{Type: TypeNotFound, Message: msg}
}

func NewInternal(msg string, err error) *AppError {
    return &AppError{Type: TypeInternal, Message: msg, Err: err}
}