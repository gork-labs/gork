package gorkson

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// SetFieldValueFromString converts a string value (for example an HTTP parameter)
// to the type of fieldValue and sets it. A registered codec has priority over the
// basic conversion of strings, numbers, booleans, string slices and pointers.
func SetFieldValueFromString(ctx context.Context, fieldValue reflect.Value, value string) error {
	fieldType := fieldValue.Type()

	if parser, exists := globalCodecRegistry.Parser(fieldType); exists {
		parsed, err := parseWithCodec(ctx, fieldType, parser, value)
		if err != nil {
			return err
		}
		fieldValue.Set(parsed)
		return nil
	}

	if fieldType.Kind() == reflect.Pointer {
		elem := reflect.New(fieldType.Elem())
		if err := SetFieldValueFromString(ctx, elem.Elem(), value); err != nil {
			return err
		}
		fieldValue.Set(elem)
		return nil
	}

	return setBasicValueFromString(fieldValue, value)
}

// setBasicValueFromString converts a string value for basic kinds and string slices.
func setBasicValueFromString(fieldValue reflect.Value, value string) error {
	kind := fieldValue.Kind()
	switch {
	case kind == reflect.String:
		fieldValue.SetString(value)
		return nil
	case isIntKind(kind):
		iv, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer value: %s", value)
		}
		fieldValue.SetInt(iv)
		return nil
	case isUintKind(kind):
		uv, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer value: %s", value)
		}
		fieldValue.SetUint(uv)
		return nil
	case kind == reflect.Bool:
		bv, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %s", value)
		}
		fieldValue.SetBool(bv)
		return nil
	case isFloatKind(kind):
		fv, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float value: %s", value)
		}
		fieldValue.SetFloat(fv)
		return nil
	case kind == reflect.Slice:
		return setStringSliceFromString(fieldValue, value)
	default:
		return fmt.Errorf("unsupported field type: %s", kind)
	}
}

// setStringSliceFromString sets a string slice from a comma-separated value.
func setStringSliceFromString(fieldValue reflect.Value, value string) error {
	if fieldValue.Type().Elem().Kind() != reflect.String {
		return fmt.Errorf("only string slices are supported in query/path/header parameters")
	}

	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	sliceVal := reflect.MakeSlice(fieldValue.Type(), len(parts), len(parts))
	for idx, part := range parts {
		sliceVal.Index(idx).SetString(strings.TrimSpace(part))
	}
	fieldValue.Set(sliceVal)
	return nil
}

// FormatFieldValueToString converts a value to its string form (for example for an
// HTTP header). A registered codec has priority. Basic kinds use their plain text form,
// a nil value gives an empty string, and other values give their JSON encoding.
func FormatFieldValueToString(ctx context.Context, value any) (string, error) {
	return formatValueToString(ctx, reflect.ValueOf(value))
}

func formatValueToString(ctx context.Context, val reflect.Value) (string, error) {
	if !val.IsValid() {
		return "", nil
	}

	if formatter, exists := globalCodecRegistry.Formatter(val.Type()); exists {
		return formatWithCodec(ctx, val, formatter)
	}

	kind := val.Kind()
	switch {
	case kind == reflect.Pointer:
		if val.IsNil() {
			return "", nil
		}
		return formatValueToString(ctx, val.Elem())
	case kind == reflect.String:
		return val.String(), nil
	case isIntKind(kind):
		return strconv.FormatInt(val.Int(), 10), nil
	case isUintKind(kind):
		return strconv.FormatUint(val.Uint(), 10), nil
	case kind == reflect.Bool:
		return strconv.FormatBool(val.Bool()), nil
	case isFloatKind(kind):
		return strconv.FormatFloat(val.Float(), 'g', -1, 64), nil
	default:
		data, err := json.Marshal(val.Interface())
		return string(data), err
	}
}

func isIntKind(kind reflect.Kind) bool {
	return kind == reflect.Int || kind == reflect.Int8 || kind == reflect.Int16 || kind == reflect.Int32 || kind == reflect.Int64
}

func isUintKind(kind reflect.Kind) bool {
	return kind == reflect.Uint || kind == reflect.Uint8 || kind == reflect.Uint16 || kind == reflect.Uint32 || kind == reflect.Uint64
}

func isFloatKind(kind reflect.Kind) bool {
	return kind == reflect.Float32 || kind == reflect.Float64
}
