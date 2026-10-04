// Package gorkson provides JSON marshaling/unmarshaling using gork tags and
// the type codec registry that pkg/api uses for all value conversion.
package gorkson

import (
	"encoding/json"
	"maps"
	"reflect"
	"strings"
)

// Marshaler handles JSON marshaling using gork tags only.
type Marshaler struct{}

// MarshalToJSON marshals a struct using gork tags for field names.
func (m *Marshaler) MarshalToJSON(v any) ([]byte, error) {
	// Check if the value implements json.Marshaler interface
	if marshaler, ok := v.(json.Marshaler); ok {
		// Use the standard JSON marshaling interface directly
		return marshaler.MarshalJSON()
	}

	converted, err := m.convertToGorkSON(v)
	if err != nil {
		return nil, err
	}
	return json.Marshal(converted)
}

// UnmarshalFromJSON unmarshals JSON into a struct using gork tags for field names.
func (m *Marshaler) UnmarshalFromJSON(data []byte, v any) error {
	// Check if the value implements json.Unmarshaler interface
	if unmarshaler, ok := v.(json.Unmarshaler); ok {
		// Use the standard JSON unmarshaling interface directly
		return unmarshaler.UnmarshalJSON(data)
	}

	// First unmarshal into a map
	var jsonMap map[string]any
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		return err
	}

	// Convert the map back to the struct using gork tag mapping
	return m.convertFromGorkSON(jsonMap, v)
}

// convertToGorkSON converts a value to a JSON-ready value using gork tags for field names.
func (m *Marshaler) convertToGorkSON(v any) (any, error) {
	return m.convertValueToGorkSON(reflect.ValueOf(v))
}

// convertValueToGorkSON converts a reflected value, formatting codec types with their codec
// and json.Marshaler types with their MarshalJSON method.
func (m *Marshaler) convertValueToGorkSON(val reflect.Value) (any, error) {
	if !val.IsValid() {
		return nil, nil
	}

	if formatter, exists := globalCodecRegistry.Formatter(val.Type()); exists {
		return formatJSONWithCodec(val, formatter)
	}

	kind := val.Kind()
	if kind == reflect.Pointer || kind == reflect.Interface {
		if val.IsNil() {
			return nil, nil
		}
		return m.convertValueToGorkSON(val.Elem())
	}
	if marshaler, ok := val.Interface().(json.Marshaler); ok {
		data, err := marshaler.MarshalJSON()
		return json.RawMessage(data), err
	}
	if kind == reflect.Slice {
		return m.convertSliceToGorkSON(val)
	}
	if kind == reflect.Struct {
		return m.convertStructToGorkSON(val)
	}
	return val.Interface(), nil
}

// convertSliceToGorkSON converts a slice to gorkson format.
func (m *Marshaler) convertSliceToGorkSON(val reflect.Value) ([]any, error) {
	result := make([]any, val.Len())
	for i := 0; i < val.Len(); i++ {
		item, err := m.convertValueToGorkSON(val.Index(i))
		if err != nil {
			return nil, err
		}
		result[i] = item
	}
	return result, nil
}

// convertStructToGorkSON converts a struct to gorkson format using field tags.
func (m *Marshaler) convertStructToGorkSON(val reflect.Value) (map[string]any, error) {
	result := make(map[string]any)
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		// Skip unexported fields
		if !fieldValue.CanInterface() {
			continue
		}

		if isEmbeddedStruct(field) {
			embedded, err := m.convertStructToGorkSON(fieldValue)
			if err != nil {
				return nil, err
			}
			maps.Copy(result, embedded)
			continue
		}

		fieldName := m.getFieldName(field)
		if fieldName == "-" {
			continue
		}

		value, err := m.convertValueToGorkSON(fieldValue)
		if err != nil {
			return nil, err
		}
		result[fieldName] = value
	}

	return result, nil
}

// convertFromGorkSON converts a JSON map back to a struct using gork tag mapping.
func (m *Marshaler) convertFromGorkSON(jsonMap map[string]any, v any) error {
	val := reflect.ValueOf(v)
	if !m.isStructPointer(val) {
		return m.convertNonStruct(jsonMap, v)
	}

	structVal := val.Elem()
	fieldMap := m.buildFieldMap(structVal.Type())
	return m.setFieldsFromMap(structVal, fieldMap, jsonMap)
}

// isStructPointer checks if the value is a pointer to a struct.
func (m *Marshaler) isStructPointer(val reflect.Value) bool {
	return val.Kind() == reflect.Pointer && val.Elem().Kind() == reflect.Struct
}

// convertNonStruct handles non-struct types using standard JSON unmarshaling.
func (m *Marshaler) convertNonStruct(jsonMap map[string]any, v any) error {
	data, err := json.Marshal(jsonMap)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// buildFieldMap maps the JSON name of each field to the index path of the field.
func (m *Marshaler) buildFieldMap(structType reflect.Type) map[string][]int {
	fieldMap := make(map[string][]int)
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if !field.IsExported() {
			continue
		}
		if isEmbeddedStruct(field) {
			for name, index := range m.buildFieldMap(field.Type) {
				fieldMap[name] = append([]int{i}, index...)
			}
			continue
		}
		if fieldName := m.getFieldName(field); fieldName != "-" {
			fieldMap[fieldName] = []int{i}
		}
	}
	return fieldMap
}

// setFieldsFromMap sets struct field values from the JSON map.
func (m *Marshaler) setFieldsFromMap(structVal reflect.Value, fieldMap map[string][]int, jsonMap map[string]any) error {
	for jsonKey, jsonValue := range jsonMap {
		if index, exists := fieldMap[jsonKey]; exists {
			if err := m.setFieldValue(structVal.FieldByIndex(index), jsonValue); err != nil {
				return err
			}
		}
	}
	return nil
}

// getFieldName returns the JSON name of the field: the gork tag name, else the
// json tag name, else the Go field name, as in encoding/json. The name "-"
// means that the field is not encoded.
func (m *Marshaler) getFieldName(field reflect.StructField) string {
	name := parseGorkTag(field.Tag.Get("gork")).Name
	if name == "" {
		name = strings.Split(field.Tag.Get("json"), ",")[0]
	}
	if name == "" {
		return field.Name
	}
	return name
}

// isEmbeddedStruct reports whether the fields of the embedded struct field are
// encoded as fields of the outer struct, as in encoding/json.
func isEmbeddedStruct(field reflect.StructField) bool {
	return field.Anonymous && field.Type.Kind() == reflect.Struct && field.Tag.Get("gork") == "" && field.Tag.Get("json") == ""
}

// GorkTagInfo represents parsed information from a gork struct tag.
type GorkTagInfo struct {
	Name string
}

// parseGorkTag parses a gork struct tag and returns the tag information.
func parseGorkTag(tag string) GorkTagInfo {
	if tag == "" {
		return GorkTagInfo{}
	}

	// Split by comma to handle multiple options (e.g., "fieldName,discriminator=value")
	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])

	return GorkTagInfo{
		Name: name,
	}
}

// setFieldValue sets a reflect.Value from an interface{} value.
func (m *Marshaler) setFieldValue(field reflect.Value, value any) error {
	if value == nil {
		return nil
	}

	if parser, exists := globalCodecRegistry.Parser(field.Type()); exists {
		return setJSONValueWithCodec(field, parser, value)
	}

	kind := field.Kind()

	// Check if it's a basic field type
	if m.isBasicFieldKind(kind) {
		return m.setBasicFieldValue(field, kind, value)
	}

	// Handle specific non-basic types
	if kind == reflect.Struct {
		return m.setStructField(field, value)
	}
	if kind == reflect.Pointer {
		return m.setPtrField(field, value)
	}
	if items, isArray := value.([]any); isArray && kind == reflect.Slice {
		return m.setSliceField(field, items)
	}

	// Handle all other types as generic fields
	return m.setGenericField(field, value)
}

// isBasicFieldKind checks if the kind is a basic type that can be set directly.
func (m *Marshaler) isBasicFieldKind(kind reflect.Kind) bool {
	return kind == reflect.String ||
		kind == reflect.Int || kind == reflect.Int8 || kind == reflect.Int16 || kind == reflect.Int32 || kind == reflect.Int64 ||
		kind == reflect.Uint || kind == reflect.Uint8 || kind == reflect.Uint16 || kind == reflect.Uint32 || kind == reflect.Uint64 ||
		kind == reflect.Float32 || kind == reflect.Float64 ||
		kind == reflect.Bool
}

// setBasicFieldValue handles setting basic field types.
func (m *Marshaler) setBasicFieldValue(field reflect.Value, kind reflect.Kind, value any) error {
	if kind == reflect.String {
		return m.setStringField(field, value)
	}
	if kind == reflect.Int || kind == reflect.Int8 || kind == reflect.Int16 || kind == reflect.Int32 || kind == reflect.Int64 {
		return m.setIntField(field, value)
	}
	if kind == reflect.Uint || kind == reflect.Uint8 || kind == reflect.Uint16 || kind == reflect.Uint32 || kind == reflect.Uint64 {
		return m.setUintField(field, value)
	}
	if kind == reflect.Float32 || kind == reflect.Float64 {
		return m.setFloatField(field, value)
	}
	if kind == reflect.Bool {
		return m.setBoolField(field, value)
	}
	// Should not reach here if isBasicFieldKind is correct
	return nil
}

// setStringField sets a string field value.
func (m *Marshaler) setStringField(field reflect.Value, value any) error {
	if str, ok := value.(string); ok {
		field.SetString(str)
	}
	return nil
}

// setIntField sets an integer field value.
func (m *Marshaler) setIntField(field reflect.Value, value any) error {
	if num, ok := value.(float64); ok {
		field.SetInt(int64(num))
	}
	return nil
}

// setUintField sets an unsigned integer field value.
func (m *Marshaler) setUintField(field reflect.Value, value any) error {
	if num, ok := value.(float64); ok {
		field.SetUint(uint64(num))
	}
	return nil
}

// setFloatField sets a float field value.
func (m *Marshaler) setFloatField(field reflect.Value, value any) error {
	if num, ok := value.(float64); ok {
		field.SetFloat(num)
	}
	return nil
}

// setBoolField sets a boolean field value.
func (m *Marshaler) setBoolField(field reflect.Value, value any) error {
	if b, ok := value.(bool); ok {
		field.SetBool(b)
	}
	return nil
}

// setStructField sets a struct field value.
func (m *Marshaler) setStructField(field reflect.Value, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	newVal := reflect.New(field.Type())
	if err := m.UnmarshalFromJSON(data, newVal.Interface()); err != nil {
		return err
	}
	field.Set(newVal.Elem())
	return nil
}

// setPtrField sets a pointer field value.
func (m *Marshaler) setPtrField(field reflect.Value, value any) error {
	elem := reflect.New(field.Type().Elem())
	if err := m.setFieldValue(elem.Elem(), value); err != nil {
		return err
	}
	field.Set(elem)
	return nil
}

// setSliceField sets a slice field item by item from a JSON array.
func (m *Marshaler) setSliceField(field reflect.Value, items []any) error {
	slice := reflect.MakeSlice(field.Type(), len(items), len(items))
	for i, item := range items {
		if err := m.setFieldValue(slice.Index(i), item); err != nil {
			return err
		}
	}
	field.Set(slice)
	return nil
}

// setGenericField sets a generic field value using JSON marshaling.
func (m *Marshaler) setGenericField(field reflect.Value, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	newVal := reflect.New(field.Type())
	if err := json.Unmarshal(data, newVal.Interface()); err != nil {
		return err
	}
	field.Set(newVal.Elem())
	return nil
}

// Global instance for convenience.
var defaultMarshaler = &Marshaler{}

// Marshal marshals using gork tags.
func Marshal(v any) ([]byte, error) {
	return defaultMarshaler.MarshalToJSON(v)
}

// Unmarshal unmarshals using gork tags.
func Unmarshal(data []byte, v any) error {
	return defaultMarshaler.UnmarshalFromJSON(data, v)
}
