package a

import "github.com/gork-labs/gork/pkg/api"

type UploadRequest struct {
	Path struct {
		ChatID string `gork:"chat_id"`
	}
	Body struct {
		Text   string     `gork:"text"`
		Avatar api.File   `gork:"avatar"`
		Images []api.File `gork:"images"`
	}
}

type BadUploadRequest struct {
	Body struct {
		Images []api.File // want "field 'Body.Images' missing gork tag"
	}
}
