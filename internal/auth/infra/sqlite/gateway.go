package sqlite

import (
	"context"

	"github.com/YuukanOO/seelf/internal/auth/app/query"
	"github.com/YuukanOO/seelf/pkg/storage"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite/builder"
)

type gateway struct {
	sqlite.Database
}

func NewGateway(db sqlite.Database) query.Gateway {
	return &gateway{db}
}

func (s *gateway) GetUsersCount(ctx context.Context) (uint, error) {
	return builder.
		Query[uint]("SELECT COUNT(id) FROM users").
		Extract(s, ctx)
}

func (s *gateway) GetAllUsers(ctx context.Context) ([]query.User, error) {
	return builder.
		Query[query.User](`
			SELECT
				id
				,email
				,registered_at
			FROM users
			ORDER BY registered_at DESC`).
		All(s, ctx, userMapper)
}

func (s *gateway) GetUserByID(ctx context.Context, id string) (query.User, error) {
	return builder.
		Query[query.User](`
			SELECT
				id
				,email
				,registered_at
			FROM users
			WHERE id = ?`, id).
		One(s, ctx, userMapper)
}

func (s *gateway) GetProfile(ctx context.Context, id string) (query.Profile, error) {
	return builder.
		Query[query.Profile](`
			SELECT
				id
				,email
				,registered_at
				,api_key
			FROM users
			WHERE id = ?`, id).
		One(s, ctx, profileMapper)
}

func userMapper(row storage.Scanner) (u query.User, err error) {
	err = row.Scan(
		&u.ID,
		&u.Email,
		&u.RegisteredAt,
	)

	return u, err
}

func profileMapper(row storage.Scanner) (u query.Profile, err error) {
	err = row.Scan(
		&u.ID,
		&u.Email,
		&u.RegisteredAt,
		&u.APIKey,
	)

	return u, err
}
