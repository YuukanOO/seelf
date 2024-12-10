package sqlite

import (
	"context"

	"github.com/YuukanOO/seelf/internal/auth/app/api_login"
	"github.com/YuukanOO/seelf/internal/auth/app/get_profile"
	"github.com/YuukanOO/seelf/pkg/storage"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite/builder"
)

type Gateway struct {
	db *sqlite.Database
}

func NewGateway(db *sqlite.Database) *Gateway {
	return &Gateway{db}
}

func (s *Gateway) GetProfile(ctx context.Context, q get_profile.Query) (get_profile.Profile, error) {
	return builder.
		Query[get_profile.Profile](`
			SELECT
				id
				,email
				,registered_at
				,api_key
			FROM users
			WHERE id = ?`, q.ID).
		One(s.db, ctx, profileMapper)
}

func (s *Gateway) GetIDFromAPIKey(ctx context.Context, c api_login.Query) (string, error) {
	return builder.
		Query[string]("SELECT id FROM users WHERE api_key = ?", c.Key).
		Extract(s.db, ctx)
}

func profileMapper(row storage.Scanner) (p get_profile.Profile, err error) {
	err = row.Scan(
		&p.ID,
		&p.Email,
		&p.RegisteredAt,
		&p.APIKey,
	)

	return p, err
}
