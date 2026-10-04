package handlers

import (
	"context"
	"time"

	"github.com/gork-labs/gork/pkg/api"
)

// UserActivityRequest selects the user whose activity is streamed.
type UserActivityRequest struct {
	Path struct {
		// UserID is the ID of the user to follow
		UserID string `gork:"userId" validate:"required"`
	}
}

// UserActivity describes one action of a user.
type UserActivity struct {
	// UserID is the ID of the user who did the action
	UserID string `gork:"userId"`
	// Action is the name of the action
	Action string `gork:"action"`
}

// UserActivityEvents lists the events of the user activity stream.
type UserActivityEvents struct {
	// Activity is sent for each action of the user
	Activity *UserActivity `gork:"activity"`
	// Reset tells the client to reload its list of actions
	Reset *struct{} `gork:"reset"`
}

// StreamUserActivity sends the actions of a user as Server-Sent Events.
func StreamUserActivity(ctx context.Context, req UserActivityRequest, stream *api.Stream[UserActivityEvents]) error {
	if err := stream.Send(UserActivityEvents{Reset: &struct{}{}}); err != nil {
		return err
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := stream.Send(UserActivityEvents{Activity: &UserActivity{UserID: req.Path.UserID, Action: "viewed"}}); err != nil {
				return err
			}
		}
	}
}
