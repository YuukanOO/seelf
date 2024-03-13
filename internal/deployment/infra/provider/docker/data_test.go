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
		)

		testutil.IsTrue(t, c1.Equals(c2))
		testutil.IsFalse(t, c1.Equals(c3))
	})
}
