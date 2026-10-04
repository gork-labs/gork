package gorkson

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestTimeCodecIsRegisteredByDefault(t *testing.T) {
	schema, ok := GetCodecRegistry().Schema(reflect.TypeOf(time.Time{}))
	if !ok {
		t.Fatal("time.Time has no default codec")
	}
	if !reflect.DeepEqual(schema, TimeCodec{}.Schema()) {
		t.Errorf("time.Time schema = %+v, want the TimeCodec schema", schema)
	}
}

func TestRegisterCodec_ReplacesTimeCodec(t *testing.T) {
	useTestCodecRegistry(t)
	if err := RegisterCodec[time.Time](UnixTimeCodec{}); err != nil {
		t.Fatalf("RegisterCodec() error = %v", err)
	}

	data, err := Marshal(struct {
		At time.Time `gork:"at"`
	}{At: time.Unix(1700000000, 0)})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if want := `{"at":1700000000}`; string(data) != want {
		t.Errorf("Marshal() = %s, want %s", data, want)
	}
}

func TestRegisterCodec_TypeErasedFunctions(t *testing.T) {
	useTestCodecRegistry(t)
	if err := RegisterCodec[Code](codeCodec{}); err != nil {
		t.Fatalf("RegisterCodec() error = %v", err)
	}
	codeType := reflect.TypeOf(Code{})

	parse, _ := GetCodecRegistry().Parser(codeType)
	if _, err := parse(context.Background(), "XX"); err == nil {
		t.Error("parser expected the codec parse error")
	}
	parsed, err := parse(context.Background(), "AB")
	if err != nil || parsed.(*Code).Value != "AB" {
		t.Errorf("parser = %v, %v, want AB", parsed, err)
	}

	format, _ := GetCodecRegistry().Formatter(codeType)
	if _, err := format(context.Background(), Code{}); err == nil {
		t.Error("formatter expected an error for a value that is not *Code")
	}
}

func TestRegisterCodec_InterfaceTypeFails(t *testing.T) {
	if err := RegisterCodec[any](anyCodec{}); err == nil {
		t.Error("RegisterCodec() expected an error for an interface type")
	}
}

type anyCodec struct{}

func (anyCodec) Parse(context.Context, string) (*any, error) { return nil, nil }

func (anyCodec) Format(context.Context, *any) (string, error) { return "", nil }

func (anyCodec) Schema() OpenAPISchema { return OpenAPISchema{} }
