package unions

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/go-playground/validator/v10"
)

// ValidatorInstance is a cached validator to avoid recreation on each unmarshal.
var (
	validatorInstance *validator.Validate
	validatorOnce     sync.Once
)

func getValidator() *validator.Validate {
	validatorOnce.Do(func() {
		validatorInstance = validator.New()
	})
	return validatorInstance
}

// Union2 represents a union of two types.
type Union2[A, B any] struct {
	A *A
	B *B
}

// UnmarshalJSON implements json.Unmarshaler for Union2.
func (u *Union2[A, B]) UnmarshalJSON(data []byte) error {
	u.A = nil
	u.B = nil

	// Try unmarshaling in order, with validation
	validate := getValidator()

	// Try type A first
	var a A
	if err := json.Unmarshal(data, &a); err == nil {
		if err := validate.Struct(a); err == nil {
			u.A = &a
			return nil
		}
	}

	// Try type B
	var b B
	if err := json.Unmarshal(data, &b); err == nil {
		if err := validate.Struct(b); err == nil {
			u.B = &b
			return nil
		}
	}

	return fmt.Errorf("failed to unmarshal into any union type: data does not match any of the union variants")
}

// MarshalJSON implements json.Marshaler for Union2.
func (u Union2[A, B]) MarshalJSON() ([]byte, error) {
	switch {
	case u.A != nil:
		return json.Marshal(u.A)
	case u.B != nil:
		return json.Marshal(u.B)
	default:
		return nil, errors.New("no value set in union")
	}
}

// Validate validates the active union member.
func (u Union2[A, B]) Validate(validate *validator.Validate) error {
	count := 0
	var value interface{}

	if u.A != nil {
		count++
		value = u.A
	}
	if u.B != nil {
		count++
		value = u.B
	}

	if count == 0 {
		return errors.New("exactly one union option must be set")
	}
	if count > 1 {
		return errors.New("only one union option can be set")
	}

	return validate.Struct(value)
}

// Value returns the active value and its type index (0-based).
func (u Union2[A, B]) Value() (interface{}, int) {
	switch {
	case u.A != nil:
		return u.A, 0
	case u.B != nil:
		return u.B, 1
	default:
		return nil, -1
	}
}

// Union3 represents a union of three types.
type Union3[A, B, C any] struct {
	A *A
	B *B
	C *C
}

// UnmarshalJSON implements json.Unmarshaler for Union3.
func (u *Union3[A, B, C]) UnmarshalJSON(data []byte) error {
	u.A = nil
	u.B = nil
	u.C = nil

	validate := getValidator()

	// Try type A first
	var a A
	if err := json.Unmarshal(data, &a); err == nil {
		if err := validate.Struct(a); err == nil {
			u.A = &a
			return nil
		}
	}

	// Try type B
	var b B
	if err := json.Unmarshal(data, &b); err == nil {
		if err := validate.Struct(b); err == nil {
			u.B = &b
			return nil
		}
	}

	// Try type C
	var c C
	if err := json.Unmarshal(data, &c); err == nil {
		if err := validate.Struct(c); err == nil {
			u.C = &c
			return nil
		}
	}

	return fmt.Errorf("failed to unmarshal into any union type: data does not match any of the union variants")
}

// MarshalJSON implements json.Marshaler for Union3.
func (u Union3[A, B, C]) MarshalJSON() ([]byte, error) {
	switch {
	case u.A != nil:
		return json.Marshal(u.A)
	case u.B != nil:
		return json.Marshal(u.B)
	case u.C != nil:
		return json.Marshal(u.C)
	default:
		return nil, errors.New("no value set in union")
	}
}

// Validate validates the active union member.
func (u Union3[A, B, C]) Validate(validate *validator.Validate) error {
	count := 0
	var value interface{}

	if u.A != nil {
		count++
		value = u.A
	}
	if u.B != nil {
		count++
		value = u.B
	}
	if u.C != nil {
		count++
		value = u.C
	}

	if count == 0 {
		return errors.New("exactly one union option must be set")
	}
	if count > 1 {
		return errors.New("only one union option can be set")
	}

	return validate.Struct(value)
}

// Value returns the active value and its type index (0-based).
func (u Union3[A, B, C]) Value() (interface{}, int) {
	switch {
	case u.A != nil:
		return u.A, 0
	case u.B != nil:
		return u.B, 1
	case u.C != nil:
		return u.C, 2
	default:
		return nil, -1
	}
}

// Union4 represents a union of four types.
type Union4[A, B, C, D any] struct {
	A *A
	B *B
	C *C
	D *D
}

// UnmarshalJSON implements json.Unmarshaler for Union4.
func (u *Union4[A, B, C, D]) UnmarshalJSON(data []byte) error {
	u.A = nil
	u.B = nil
	u.C = nil
	u.D = nil

	validate := getValidator()

	// Try type A first
	var a A
	if err := json.Unmarshal(data, &a); err == nil {
		if err := validate.Struct(a); err == nil {
			u.A = &a
			return nil
		}
	}

	// Try type B
	var b B
	if err := json.Unmarshal(data, &b); err == nil {
		if err := validate.Struct(b); err == nil {
			u.B = &b
			return nil
		}
	}

	// Try type C
	var c C
	if err := json.Unmarshal(data, &c); err == nil {
		if err := validate.Struct(c); err == nil {
			u.C = &c
			return nil
		}
	}

	// Try type D
	var d D
	if err := json.Unmarshal(data, &d); err == nil {
		if err := validate.Struct(d); err == nil {
			u.D = &d
			return nil
		}
	}

	return fmt.Errorf("failed to unmarshal into any union type: data does not match any of the union variants")
}

// MarshalJSON implements json.Marshaler for Union4.
func (u Union4[A, B, C, D]) MarshalJSON() ([]byte, error) {
	switch {
	case u.A != nil:
		return json.Marshal(u.A)
	case u.B != nil:
		return json.Marshal(u.B)
	case u.C != nil:
		return json.Marshal(u.C)
	case u.D != nil:
		return json.Marshal(u.D)
	default:
		return nil, errors.New("no value set in union")
	}
}

// Validate validates the active union member.
func (u Union4[A, B, C, D]) Validate(validate *validator.Validate) error {
	count := 0
	var value interface{}

	if u.A != nil {
		count++
		value = u.A
	}
	if u.B != nil {
		count++
		value = u.B
	}
	if u.C != nil {
		count++
		value = u.C
	}
	if u.D != nil {
		count++
		value = u.D
	}

	if count == 0 {
		return errors.New("exactly one union option must be set")
	}
	if count > 1 {
		return errors.New("only one union option can be set")
	}

	return validate.Struct(value)
}

// Value returns the active value and its type index (0-based).
func (u Union4[A, B, C, D]) Value() (interface{}, int) {
	switch {
	case u.A != nil:
		return u.A, 0
	case u.B != nil:
		return u.B, 1
	case u.C != nil:
		return u.C, 2
	case u.D != nil:
		return u.D, 3
	default:
		return nil, -1
	}
}
