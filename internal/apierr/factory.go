package apierr

func NewValidation(msg string) *AppError {
    return &AppError{Type: TypeValidation, Message: msg}
}

func NewNotFound(msg string) *AppError {
    return &AppError{Type: TypeNotFound, Message: msg}
}

func NewConflict(msg string) *AppError {
	return &AppError{Type: TypeConflict, Message: msg}
}

func NewUnauthorized(msg string) *AppError {
	return &AppError{Type: TypeUnauthorized, Message: msg}
}

func NewForbidden(msg string) *AppError {
	return &AppError{Type: TypeForbidden, Message: msg}
}

func NewInternal(msg string, err error) *AppError {
	return &AppError{Type: TypeInternal, Message: msg, Err: err}
}