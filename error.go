package hood

const (
	ValidationErrorValueNotSet = 0x8000 + iota
	ValidationErrorValueTooSmall
	ValidationErrorValueTooBig
	ValidationErrorValueTooShort
	ValidationErrorValueTooLong
)

// Validation error type
type ValidationError struct {
	id int
	s  string
}

// NewValidationError returns a new validation error with the specified id and
// text. The id's purpose is to distinguish different validation error types.
// Built-in validation error ids start at 0x8000.
func NewValidationError(id int, text string) error {
	return &ValidationError{id, text}
}

func (e *ValidationError) Error() string {
	return e.s
}

func (e *ValidationError) Id() int {
	return e.id
}
