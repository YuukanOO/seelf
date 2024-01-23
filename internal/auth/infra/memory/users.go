package memory

import (
	"context"
	"errors"
	"slices"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/event"
)

type (
	UsersStore interface {
		domain.UsersReader
		domain.UsersWriter
	}

	usersStore struct {
		users []*userData
	}

	userData struct {
		id    domain.UserID
		key   domain.APIKey
		email domain.Email
		value *domain.User
	}
)

func NewUsersStore(existingUsers ...*domain.User) UsersStore {
	s := &usersStore{}

	s.Write(context.Background(), existingUsers...)

	return s
}

func (s *usersStore) GetUsersCount(ctx context.Context) (uint, error) {
	return uint(len(s.users)), nil
}

func (s *usersStore) GetEmailAvailability(ctx context.Context, email domain.Email, excluded ...domain.UserID) (domain.EmailAvailability, error) {
	u, err := s.GetByEmail(ctx, email)

	if errors.Is(err, apperr.ErrNotFound) || slices.Contains(excluded, u.ID()) {
		return true, nil
	}

	if err != nil {
		return false, err
	}

	return false, nil
}

func (s *usersStore) GetByID(ctx context.Context, id domain.UserID) (domain.User, error) {
	for _, u := range s.users {
		if u.id == id {
			return *u.value, nil
		}
	}

	return domain.User{}, apperr.ErrNotFound
}

func (s *usersStore) GetByEmail(ctx context.Context, email domain.Email) (domain.User, error) {
	for _, u := range s.users {
		if u.email == email {
			return *u.value, nil
		}
	}

	return domain.User{}, apperr.ErrNotFound
}

func (s *usersStore) UsersCount(ctx context.Context) (uint, error) {
	return uint(len(s.users)), nil
}

func (s *usersStore) GetIDFromAPIKey(ctx context.Context, key domain.APIKey) (domain.UserID, error) {
	for _, u := range s.users {
		if u.key == key {
			return u.id, nil
		}
	}

	return "", apperr.ErrNotFound
}

func (s *usersStore) Write(ctx context.Context, users ...*domain.User) error {
	for _, user := range users {
		for _, e := range event.Unwrap(user) {
			switch evt := e.(type) {
			case domain.UserRegistered:
				var exist bool
				for _, a := range s.users {
					if a.id == evt.ID {
						exist = true
						break
					}
				}

				if exist {
					continue
				}

				s.users = append(s.users, &userData{
					id:    evt.ID,
					email: evt.Email,
					key:   evt.Key,
					value: user,
				})
			default:
				for _, u := range s.users {
					if u.id == user.ID() {
						*u.value = *user
						break
					}
				}
			}
		}
	}

	return nil
}
