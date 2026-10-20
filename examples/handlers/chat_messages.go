package handlers

import (
	"context"

	"github.com/gork-labs/gork/pkg/api"
)

// Mention is a user that a message mentions.
type Mention struct {
	// UserID is the user that the message mentions
	UserID string `gork:"user_id" validate:"required,uuid"`

	// Offset is the position of the mention in the text of the message
	Offset int `gork:"offset" validate:"min=0"`
}

// SendMessageRequest represents a chat message with images.
// The client sends it as multipart/form-data.
type SendMessageRequest struct {
	Path struct {
		// ChatID is the chat that gets the message
		ChatID string `gork:"chat_id" validate:"required,uuid"`
	}
	Body struct {
		// Text is the text of the message
		Text string `gork:"text" validate:"required,max=4000"`

		// Images are the images of the message
		Images []api.File `gork:"images" validate:"max=4"`

		// Mentions are the users that the message mentions
		Mentions []Mention `gork:"mentions" validate:"max=20,dive"`
	}
}

// SendMessageResponse represents the message that the chat got.
type SendMessageResponse struct {
	Body struct {
		// Text is the text of the message
		Text string `gork:"text"`

		// ImageNames are the file names of the images of the message
		ImageNames []string `gork:"image_names"`

		// MentionUserIDs are the users that the message mentions
		MentionUserIDs []string `gork:"mention_user_ids"`
	}
}

// SendMessage adds a message with images to a chat.
func SendMessage(_ context.Context, req SendMessageRequest) (*SendMessageResponse, error) {
	resp := &SendMessageResponse{}
	resp.Body.Text = req.Body.Text
	resp.Body.ImageNames = []string{}
	for _, image := range req.Body.Images {
		resp.Body.ImageNames = append(resp.Body.ImageNames, image.Name)
	}
	resp.Body.MentionUserIDs = []string{}
	for _, mention := range req.Body.Mentions {
		resp.Body.MentionUserIDs = append(resp.Body.MentionUserIDs, mention.UserID)
	}
	return resp, nil
}
