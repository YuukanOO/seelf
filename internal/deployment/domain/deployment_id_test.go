package domain_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_DeploymentID(t *testing.T) {
	t.Run("could be created from an appid and a deployment number", func(t *testing.T) {
		var (
			app    domain.AppID            = "1"
			number domain.DeploymentNumber = 1
		)

		id := domain.DeploymentIDFrom(app, number)

		testutil.Equals(t, app, id.AppID())
		testutil.Equals(t, number, id.DeploymentNumber())
	})
}
