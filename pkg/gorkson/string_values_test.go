package gorkson

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSetFieldValueFromString_BasicKinds(t *testing.T) {
	tests := []struct {
		name  string
		ptr   any
		value string
		want  any
	}{
		{"string", new(string), "hello", "hello"},
		{"int", new(int), "-12", -12},
		{"int8", new(int8), "12", int8(12)},
		{"uint", new(uint), "12", uint(12)},
		{"uint64", new(uint64), "12", uint64(12)},
		{"bool", new(bool), "true", true},
		{"float32", new(float32), "1.5", float32(1.5)},
		{"float64", new(float64), "2.5", 2.5},
		{"string slice", new([]string), "a, b,c", []string{"a", "b", "c"}},
		{"empty string slice", new([]string), "", []string(nil)},
		{"pointer to int", new(*int), "7", 7},
		{"time", new(time.Time), "2024-01-02T03:04:05+02:00", time.Date(2024, 1, 2, 1, 4, 5, 0, time.UTC)},
		{"pointer to time", new(*time.Time), "2024-01-02T03:04:05Z", time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.ValueOf(tt.ptr).Elem()
			if err := SetFieldValueFromString(context.Background(), field, tt.value); err != nil {
				t.Fatalf("SetFieldValueFromString() error = %v", err)
			}

			got := field
			if got.Kind() == reflect.Pointer {
				got = got.Elem()
			}
			if want, ok := tt.want.(time.Time); ok {
				if !got.Interface().(time.Time).Equal(want) {
					t.Errorf("SetFieldValueFromString() = %v, want %v", got.Interface(), want)
				}
				return
			}
			if !reflect.DeepEqual(got.Interface(), tt.want) {
				t.Errorf("SetFieldValueFromString() = %#v, want %#v", got.Interface(), tt.want)
			}
		})
	}
}

func TestSetFieldValueFromString_Errors(t *testing.T) {
	useTestCodecRegistry(t)
	if err := RegisterCodec[Code](codeCodec{}); err != nil {
		t.Fatalf("RegisterCodec() error = %v", err)
	}

	tests := []struct {
		name   string
		ptr    any
		value  string
		errMsg string
	}{
		{"invalid integer", new(int), "x", "invalid integer value"},
		{"invalid unsigned integer", new(uint), "-1", "invalid unsigned integer value"},
		{"invalid boolean", new(bool), "x", "invalid boolean value"},
		{"invalid float", new(float64), "1e999", "invalid float value"},
		{"non-string slice", new([]int), "1,2", "only string slices are supported"},
		{"unsupported type", new(map[string]string), "x", "unsupported field type"},
		{"invalid pointer value", new(*int), "x", "invalid integer value"},
		{"invalid time", new(time.Time), "2024-01-02", "failed to parse time.Time"},
		{"schema validation", new(Code), "abc", "does not match required pattern"},
		{"codec parse error", new(Code), "XX", "reserved code"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.ValueOf(tt.ptr).Elem()
			err := SetFieldValueFromString(context.Background(), field, tt.value)
			if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("SetFieldValueFromString() error = %v, want it to contain %q", err, tt.errMsg)
			}
		})
	}
}

func TestSetFieldValueFromString_Codec(t *testing.T) {
	useTestCodecRegistry(t)
	if err := RegisterCodec[Code](codeCodec{}); err != nil {
		t.Fatalf("RegisterCodec() error = %v", err)
	}

	var code Code
	if err := SetFieldValueFromString(context.Background(), reflect.ValueOf(&code).Elem(), "AB"); err != nil {
		t.Fatalf("SetFieldValueFromString() error = %v", err)
	}
	if code.Value != "AB" {
		t.Errorf("code = %q, want AB", code.Value)
	}
}

func TestFormatFieldValueToString(t *testing.T) {
	useTestCodecRegistry(t)
	if err := RegisterCodec[Level](levelCodec{}); err != nil {
		t.Fatalf("RegisterCodec() error = %v", err)
	}

	at := time.Date(2024, 1, 2, 3, 4, 5, 6, time.UTC)
	var nilTime *time.Time

	tests := []struct {
		name  string
		value any
		want  string
	}{
		{"nil", nil, ""},
		{"nil pointer", nilTime, ""},
		{"string", "hello", "hello"},
		{"int", int16(-3), "-3"},
		{"uint", uint8(3), "3"},
		{"bool", false, "false"},
		{"float32", float32(3.5), "3.5"},
		{"float64", 2.25, "2.25"},
		{"time", at, "2024-01-02T03:04:05Z"},
		{"pointer to time", &at, "2024-01-02T03:04:05Z"},
		{"codec", Level{Value: 4}, "4"},
		{"struct", struct{ Name string }{"x"}, `{"Name":"x"}`},
		{"slice", []string{"a"}, `["a"]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FormatFieldValueToString(context.Background(), tt.value)
			if err != nil {
				t.Fatalf("FormatFieldValueToString() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("FormatFieldValueToString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatFieldValueToString_Errors(t *testing.T) {
	useTestCodecRegistry(t)
	if err := RegisterCodec[Level](levelCodec{}); err != nil {
		t.Fatalf("RegisterCodec() error = %v", err)
	}

	if _, err := FormatFieldValueToString(context.Background(), Level{Value: -1}); err == nil {
		t.Error("FormatFieldValueToString() expected codec format error")
	}
	if _, err := FormatFieldValueToString(context.Background(), make(chan int)); err == nil {
		t.Error("FormatFieldValueToString() expected JSON encoding error")
	}
}
