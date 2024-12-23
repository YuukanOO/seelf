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
			,version
		FROM [auth.users]
		ORDER BY registered_at ASC
		LIMIT 1`).
		One(s.db, ctx, domain.UserFrom)
}

func (s *usersStore) CheckEmailAvailability(ctx context.Context, email domain.Email, excluded ...domain.UserID) (domain.EmailRequirement, error) {
	unique, err := builder.
		Query[bool]("SELECT NOT EXISTS(SELECT 1 FROM [auth.users] WHERE email = ?", email).
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
				,version
			FROM [auth.users]
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
				,version
			FROM [auth.users]
			WHERE email = ?`, email).
		One(s.db, ctx, domain.UserFrom)
}

func (s *usersStore) Write(c context.Context, users ...*domain.User) error {
	return sqlite.WriteEvents(s.db, c, users,
		"[auth.users]",
		func(u *domain.User) sqlite.Key {
			return sqlite.Key{
				"id": u.ID(),
			}
		},
		func(e event.Event, v builder.Values) sqlite.WriteMode {
			switch evt := e.(type) {
			case domain.UserRegistered:
				v["id"] = evt.ID
				v["email"] = evt.Email
				v["password_hash"] = evt.Password
				v["api_key"] = evt.Key
				v["registered_at"] = evt.RegisteredAt
			case domain.UserEmailChanged:
				v["email"] = evt.Email
			case domain.UserPasswordChanged:
				v["password_hash"] = evt.Password
			case domain.UserAPIKeyChanged:
				v["api_key"] = evt.Key
			}

			return sqlite.WriteModeUpsert
		})
}
