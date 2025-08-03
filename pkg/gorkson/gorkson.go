// Package gorkson provides JSON marshaling/unmarshaling using gork tags.
package gorkson

import (
	"encoding/json"
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

	return json.Marshal(m.convertToGorkSON(v))
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

// convertToGorkSON converts a struct to a map using gork tags for field names.
func (m *Marshaler) convertToGorkSON(v any) any {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	// Handle slices by converting each element
	if val.Kind() == reflect.Slice {
		result := make([]interface{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			result[i] = m.convertToGorkSON(val.Index(i).Interface())
		}
		return result
	}

	if val.Kind() != reflect.Struct {
		return v
	}

	result := make(map[string]interface{})
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		// Skip unexported fields
		if !fieldValue.CanInterface() {
			continue
		}

		// Get field name from gork tag
		fieldName := m.getFieldName(field)
		if fieldName == "" || fieldName == "-" {
			continue
		}

		// Recursively convert nested structs
		value := m.convertToGorkSON(fieldValue.Interface())
		result[fieldName] = value
	}

	return result
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
	return val.Kind() == reflect.Ptr && val.Elem().Kind() == reflect.Struct
}

// convertNonStruct handles non-struct types using standard JSON unmarshaling.
func (m *Marshaler) convertNonStruct(jsonMap map[string]any, v any) error {
	data, err := json.Marshal(jsonMap)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// buildFieldMap creates a mapping from field names to field indices.
func (m *Marshaler) buildFieldMap(structType reflect.Type) map[string]int {
	fieldMap := make(map[string]int)
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldName := m.getFieldName(field)
		if fieldName != "" && fieldName != "-" {
			fieldMap[fieldName] = i
		}
	}
	return fieldMap
}

// setFieldsFromMap sets struct field values from the JSON map.
func (m *Marshaler) setFieldsFromMap(structVal reflect.Value, fieldMap map[string]int, jsonMap map[string]any) error {
	for jsonKey, jsonValue := range jsonMap {
		if fieldIndex, exists := fieldMap[jsonKey]; exists {
			field := structVal.Field(fieldIndex)
			if field.CanSet() {
				if err := m.setFieldValue(field, jsonValue); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// getFieldName extracts the field name from gork tag, falling back to json tag if no gork tag.
func (m *Marshaler) getFieldName(field reflect.StructField) string {
	// Prefer gork tag
	if gorkTag := field.Tag.Get("gork"); gorkTag != "" {
		tagInfo := parseGorkTag(gorkTag)
		if tagInfo.Name != "" {
			return tagInfo.Name
		}
	}

	// Fall back to json tag if no gork tag found
	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		// Parse json tag (take first part before comma)
		name := strings.Split(jsonTag, ",")[0]
		if name != "" && name != "-" {
			return name
		}
	}

	// No tags found - skip field
	return ""
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

	kind := field.Kind()

	// Check if it's a basic field type
	if m.isBasicFieldKind(kind) {
		return m.setBasicFieldValue(field, kind, value)
	}

	// Handle specific non-basic types
	if kind == reflect.Struct {
		return m.setStructField(field, value)
	}
	if kind == reflect.Ptr {
		return m.setPtrField(field, value)
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
	if field.Type().Elem().Kind() == reflect.Struct {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		newVal := reflect.New(field.Type().Elem())
		if err := m.UnmarshalFromJSON(data, newVal.Interface()); err != nil {
			return err
		}
		field.Set(newVal)
	}
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
