package app

import "github.com/YuukanOO/seelf/pkg/monad"

type (
	UserSummary struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}

	TargetSummary struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Url  string `json:"url"`
	}

	LatestDeployments[T any] struct {
		Production monad.Maybe[T] `json:"production"`
		Staging    monad.Maybe[T] `json:"staging"`
	}
)
