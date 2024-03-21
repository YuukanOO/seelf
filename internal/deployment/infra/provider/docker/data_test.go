package docker_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/provider/docker"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/ssh"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Data(t *testing.T) {
	t.Run("should be comparable", func(t *testing.T) {
		var (
			c1 domain.ProviderConfig = docker.Data{
				Host: monad.Value[ssh.Host]("testdata"),
				User: monad.Value("test"),
			}
			c2 domain.ProviderConfig = docker.Data{
				Host: monad.Value[ssh.Host]("testdata"),
				User: monad.Value("test"),
			}
			c3 domain.ProviderConfig = docker.Data{
				Host: monad.Value[ssh.Host]("testdata"),
			}
			c4 domain.ProviderConfig = otherDataSameProperties{
				Host: monad.Value[ssh.Host]("testdata"),
				User: monad.Value("test"),
			}
		)

		testutil.IsTrue(t, c1.Equals(c2))
		testutil.IsFalse(t, c1.Equals(c3))
		testutil.IsFalse(t, c2.Equals(c4))
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
