package domain

// Represents basic credentials used by a registry.
type Credentials struct {
	username string
	password string
}

// Builds new credentials with the provided username and password.
func NewCredentials(username, password string) Credentials {
	return Credentials{
		username: username,
		password: password,
	}
}

// Updates the username.
func (c *Credentials) HasUsername(username string) {
	c.username = username
}

// Updates the password.
func (c *Credentials) HasPassword(password string) {
	c.password = password
}

func (c Credentials) Username() string { return c.username }
func (c Credentials) Password() string { return c.password }
