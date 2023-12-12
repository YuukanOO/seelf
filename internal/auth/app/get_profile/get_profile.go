package get_profile

import (
	"time"

	"github.com/YuukanOO/seelf/pkg/bus"
)

type (
	// Retrieve the user profile.
	Query struct {
		bus.Query[Profile]

		ID string `json:"-"`
	}

	Profile struct {
		ID           string    `json:"id"`
		Email        string    `json:"email"`
		RegisteredAt time.Time `json:"registered_at"`
		APIKey       string    `json:"api_key"`
	}
)

func (Query) Name_() string { return "auth.query.get_profile" }
