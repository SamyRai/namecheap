package errors

import "fmt"

// ErrInvalidInput represents an invalid input error
type ErrInvalidInput struct {
	Field   string
	Message string
}

func (e *ErrInvalidInput) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("invalid input %s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("invalid input: %s", e.Message)
}

// NewInvalidInput creates a new invalid input error
func NewInvalidInput(field, message string) *ErrInvalidInput {
	return &ErrInvalidInput{
		Field:   field,
		Message: message,
	}
}

// ErrNotFound represents a not found error
type ErrNotFound struct {
	Resource string
	ID       string
}

func (e *ErrNotFound) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s '%s' not found", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

// NewNotFound creates a new not found error
func NewNotFound(resource, id string) *ErrNotFound {
	return &ErrNotFound{
		Resource: resource,
		ID:       id,
	}
}

// ErrConfiguration represents a configuration error
type ErrConfiguration struct {
	Message string
}

func (e *ErrConfiguration) Error() string {
	return fmt.Sprintf("configuration error: %s", e.Message)
}

// NewConfiguration creates a new configuration error
func NewConfiguration(message string) *ErrConfiguration {
	return &ErrConfiguration{
		Message: message,
	}
}

// ErrAPI represents an API error
type ErrAPI struct {
	Operation string
	Message   string
	Err       error
}

func (e *ErrAPI) Error() string {
	// The provider's own message is the only actionable part of an API failure
	// -- Namecheap, for instance, names the offending record and why it was
	// rejected. Dropping e.Err made every failure read as an opaque
	// "API error in SetHosts", which is useless when a destructive whole-zone
	// write has just been refused.
	base := "API error"
	if e.Operation != "" {
		base = fmt.Sprintf("API error in %s", e.Operation)
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", base, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", base, e.Message)
}

func (e *ErrAPI) Unwrap() error {
	return e.Err
}

// NewAPI creates a new API error
func NewAPI(operation, message string, err error) *ErrAPI {
	return &ErrAPI{
		Operation: operation,
		Message:   message,
		Err:       err,
	}
}
