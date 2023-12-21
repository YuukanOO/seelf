package storage

// Represents a paginated data set.
type Paginated[T any] struct {
	Data        []T  `json:"data"`
	Page        int  `json:"page"`
	IsFirstPage bool `json:"first_page"`
	IsLastPage  bool `json:"last_page"`
	PerPage     int  `json:"per_page"`
	Total       int  `json:"total"`
}
