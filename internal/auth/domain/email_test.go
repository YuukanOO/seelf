package domain_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Email_ValidatesAnEmail(t *testing.T) {
	r, err := domain.EmailFrom("")
	testutil.Equals(t, "", r)
	testutil.ErrorIs(t, domain.ErrInvalidEmail, err)

	r, err = domain.EmailFrom("agood@email.com")
	testutil.Equals(t, "agood@email.com", r)
	testutil.IsNil(t, err)
}
