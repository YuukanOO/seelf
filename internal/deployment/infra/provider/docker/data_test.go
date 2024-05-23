package docker_test

import (
	"fmt"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/provider/docker"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/ssh"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Data(t *testing.T) {
	t.Run("should be comparable", func(t *testing.T) {
		tests := []struct {
			a        domain.ProviderConfig
			b        domain.ProviderConfig
			expected bool
		}{
			{
				a: docker.Data{
					Host: monad.Value[ssh.Host]("testdata"),
					User: monad.Value("test"),
				},
				b: docker.Data{
					Host: monad.Value[ssh.Host]("testdata"),
					User: monad.Value("test"),
				},
				expected: true,
			},
			{
				a: docker.Data{
					Host: monad.Value[ssh.Host]("testdata"),
					User: monad.Value("test"),
				},
				b: docker.Data{
					Host: monad.Value[ssh.Host]("testdata"),
				},
				expected: false,
			},
			{
				a: docker.Data{
					Host: monad.Value[ssh.Host]("testdata"),
					User: monad.Value("test"),
				},
				b: otherDataSameProperties{
					Host: monad.Value[ssh.Host]("testdata"),
					User: monad.Value("test"),
				},
				expected: false,
			},
		}

		for _, test := range tests {
			t.Run(fmt.Sprintf("%v", test), func(t *testing.T) {
				got := test.a.Equals(test.b)

				testutil.Equals(t, test.expected, got)
			})
		}
	})
}

type otherDataSameProperties struct {
	Host       monad.Maybe[ssh.Host]       `json:"host"`
	Port       monad.Maybe[int]            `json:"port"`
	User       monad.Maybe[string]         `json:"user"`
	PrivateKey monad.Maybe[ssh.PrivateKey] `json:"private_key"`
}

func (otherDataSameProperties) Kind() string                              { return "dummy" }
func (d otherDataSameProperties) Fingerprint() string                     { return string(d.Host.Get("")) }
func (d otherDataSameProperties) String() string                          { return d.Kind() }
func (d otherDataSameProperties) Equals(other domain.ProviderConfig) bool { return d == other }
