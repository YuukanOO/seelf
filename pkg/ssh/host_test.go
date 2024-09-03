package ssh_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/ssh"
)

func Test_Host(t *testing.T) {
	t.Run("should correctly parse a host", func(t *testing.T) {
		tests := []struct {
			value string
			valid bool
		}{
			{"2001:0db8:85a3:0000:0000:8a2e:0370:7334", true},
			{"localhost", true},
			{"localhost/example", false},
			{"localhost.example", true},
			{"localhost:3000", false},
			{"192.168.1.1", true},
			{"192.168.1.1:3000", false},
			{"192,168,1,1:3000", false},
			{"https://localhost", false},
		}

		for _, tt := range tests {
			t.Run(tt.value, func(t *testing.T) {
				got, err := ssh.ParseHost(tt.value)

				if !tt.valid {
					assert.ErrorIs(t, ssh.ErrInvalidHost, err)
					return
				}

				assert.Nil(t, err)
				assert.Equal(t, tt.value, string(got))
			})
		}
	})

	t.Run("should returns a string representation", func(t *testing.T) {
		h := must.Panic(ssh.ParseHost("localhost"))
		assert.Equal(t, "localhost", h.String())
	})
}
