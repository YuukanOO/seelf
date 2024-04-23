package sqlite

import (
	"context"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite/builder"
)

type (
	UsersStore interface {
		domain.UsersReader
		domain.UsersWriter
	}

	usersStore struct {
		db *sqlite.Database
	}
)

func NewUsersStore(db *sqlite.Database) UsersStore {
	return &usersStore{db}
}

func (s *usersStore) GetAdminUser(ctx context.Context) (domain.User, error) {
	return builder.
		Query[domain.User](`
		SELECT
			id
			,email
			,password_hash
			,api_key
			,registered_at
		FROM users
		ORDER BY registered_at ASC
		LIMIT 1`).
		One(s.db, ctx, domain.UserFrom)
}

func (s *usersStore) CheckEmailAvailability(ctx context.Context, email domain.Email, excluded ...domain.UserID) (domain.EmailRequirement, error) {
	unique, err := builder.
		Query[bool]("SELECT NOT EXISTS(SELECT 1 FROM users WHERE email = ?", email).
		S(builder.Array("AND id NOT IN", excluded)).
		F(")").
		Extract(s.db, ctx)

	return domain.NewEmailRequirement(email, unique), err
}

func (s *usersStore) GetByID(ctx context.Context, id domain.UserID) (u domain.User, err error) {
	return builder.
		Query[domain.User](`
			SELECT
				id
				,email
				,password_hash
				,api_key
				,registered_at
			FROM users
			WHERE id = ?`, id).
		One(s.db, ctx, domain.UserFrom)
}

func (s *usersStore) GetByEmail(ctx context.Context, email domain.Email) (u domain.User, err error) {
	return builder.
		Query[domain.User](`
			SELECT
				id
				,email
				,password_hash
				,api_key
				,registered_at
			FROM users
			WHERE email = ?`, email).
		One(s.db, ctx, domain.UserFrom)
}

func (s *usersStore) GetIDFromAPIKey(ctx context.Context, key domain.APIKey) (domain.UserID, error) {
	return builder.
		Query[domain.UserID]("SELECT id FROM users WHERE api_key = ?", key).
		Extract(s.db, ctx)
}

func (s *usersStore) Write(c context.Context, users ...*domain.User) error {
	return sqlite.WriteAndDispatch(s.db, c, users, func(ctx context.Context, e event.Event) error {
		switch evt := e.(type) {
		case domain.UserRegistered:
			return builder.
				Insert("users", builder.Values{
					"id":            evt.ID,
					"email":         evt.Email,
					"password_hash": evt.Password,
					"api_key":       evt.Key,
					"registered_at": evt.RegisteredAt,
				}).
				Exec(s.db, ctx)
		case domain.UserEmailChanged:
			return builder.
				Update("users", builder.Values{
					"email": evt.Email,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.UserPasswordChanged:
			return builder.
				Update("users", builder.Values{
					"password_hash": evt.Password,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.UserAPIKeyChanged:
			return builder.
				Update("users", builder.Values{
					"api_key": evt.Key,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		default:
			return nil
		}
	})
}
