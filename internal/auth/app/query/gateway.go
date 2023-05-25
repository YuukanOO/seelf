package query

import (
	"context"
	"time"
)

type (
	// Access to the underlying storage adapter for read use cases
	Gateway interface {
		GetAllUsers(context.Context) ([]User, error)
		GetUserByID(context.Context, string) (User, error)
		GetProfile(context.Context, string) (Profile, error)
	}

	User struct {
		ID           string    `json:"id"`
		Email        string    `json:"email"`
		RegisteredAt time.Time `json:"registered_at"`
	}

	Profile struct {
		User
		APIKey string `json:"api_key"`
	}
)
