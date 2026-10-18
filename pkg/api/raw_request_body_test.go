package api

import (
	"context"
	"reflect"
	"testing"
)

type uploadAudioRequest struct {
	Body []byte
}

type uploadAudioResponse struct {
	Body struct {
		Text string `json:"text"`
	}
}

func uploadAudio(_ context.Context, _ uploadAudioRequest) (*uploadAudioResponse, error) {
	return &uploadAudioResponse{}, nil
}

func TestRawRequestBodyInSpec(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		want    []string
	}{
		{
			name:    "one declared media type",
			options: []Option{WithRequestContentTypes("audio/wav")},
			want:    []string{"audio/wav"},
		},
		{
			name:    "two declared media types",
			options: []Option{WithRequestContentTypes("a", "b")},
			want:    []string{"a", "b"},
		},
		{
			name: "default media type",
			want: []string{"application/octet-stream"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRouteRegistry()
			router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
			router.Post("/transcriptions", uploadAudio, tt.options...)

			body := GenerateOpenAPI(registry).Paths["/transcriptions"].Post.RequestBody
			if body == nil || !body.Required {
				t.Fatalf("requestBody = %+v, want a required requestBody", body)
			}
			if len(body.Content) != len(tt.want) {
				t.Fatalf("content = %v, want the media types %v", body.Content, tt.want)
			}
			for _, mediaType := range tt.want {
				want := &MediaType{Schema: &Schema{Type: "string", Format: "binary"}}
				if got := body.Content[mediaType]; !reflect.DeepEqual(got, want) {
					t.Errorf("content %s = %+v, want %+v", mediaType, got, want)
				}
			}
		})
	}
}
