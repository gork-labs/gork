package gorkson

import (
	"context"
	"testing"
	"time"
)

func TestTimeCodecKeepsFractionalSeconds(t *testing.T) {
	type event struct {
		At time.Time `gork:"at"`
	}
	at := time.Date(2024, 1, 2, 3, 4, 5, 123456789, time.UTC)

	data, err := Marshal(event{At: at})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"at":"2024-01-02T03:04:05.123456789Z"}`; string(data) != want {
		t.Errorf("Marshal() = %s, want %s", data, want)
	}

	var back event
	if err := Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if !back.At.Equal(at) {
		t.Errorf("round trip = %v, want %v", back.At, at)
	}
}

func TestTimeCodecParsesWithAndWithoutFraction(t *testing.T) {
	for value, want := range map[string]time.Time{
		"2024-01-02T03:04:05Z":        time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
		"2024-01-02T03:04:05.5Z":      time.Date(2024, 1, 2, 3, 4, 5, 500000000, time.UTC),
		"2024-01-02T05:04:05.1+02:00": time.Date(2024, 1, 2, 3, 4, 5, 100000000, time.UTC),
	} {
		got, err := TimeCodec{}.Parse(context.Background(), value)
		if err != nil || !got.Equal(want) {
			t.Errorf("Parse(%q) = %v, %v, want %v", value, got, err, want)
		}
	}
}
