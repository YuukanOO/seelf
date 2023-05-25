package memory

import (
	"context"
	"errors"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/collections"
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
		email domain.UniqueEmail
		value *domain.User
	}
)

func NewUsersStore(existingUsers ...domain.User) UsersStore {
	s := &usersStore{}
	ctx := context.Background()

	s.Write(ctx, collections.ToPointers(existingUsers)...)

	return s
}

func (s *usersStore) GetUsersCount(ctx context.Context) (uint, error) {
	return uint(len(s.users)), nil
}

func (s *usersStore) IsEmailUnique(ctx context.Context, email domain.Email) (domain.UniqueEmail, error) {
	_, err := s.GetByEmail(ctx, email)

	if errors.Is(err, apperr.ErrNotFound) {
		return domain.UniqueEmail(email), nil
	}

	if err == nil {
		return "", domain.ErrEmailAlreadyTaken
	}

	return "", err
}

func (s *usersStore) IsEmailUniqueForUser(ctx context.Context, id domain.UserID, email domain.Email) (domain.UniqueEmail, error) {
	u, err := s.GetByEmail(ctx, email)

	if errors.Is(err, apperr.ErrNotFound) || u.ID() == id {
		return domain.UniqueEmail(email), nil
	}

	if err != nil {
		return "", err
	}

	return "", domain.ErrEmailAlreadyTaken
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
	unique := domain.UniqueEmail(email)

	for _, u := range s.users {
		if u.email == unique {
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
						u.value = user
						break
					}
				}
			}
		}
	}

	return nil
}
