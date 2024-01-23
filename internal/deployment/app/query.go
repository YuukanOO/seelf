package app

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
)
