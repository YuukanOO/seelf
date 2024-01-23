package domain

type EmailRequirement struct {
	email  Email
	unique bool
}

func NewEmailRequirement(email Email, unique bool) EmailRequirement {
	return EmailRequirement{
		email:  email,
		unique: unique,
	}
}

func (e EmailRequirement) Error() error {
	if !e.unique {
		return ErrEmailAlreadyTaken
	}

	return nil
}

func (e EmailRequirement) Met() (Email, error) { return e.email, e.Error() }
