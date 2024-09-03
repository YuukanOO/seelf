package domain_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/assert"
)

func Test_DeploymentID(t *testing.T) {
	t.Run("could be created from an appid and a deployment number", func(t *testing.T) {
		var (
			app    domain.AppID            = "1"
			number domain.DeploymentNumber = 1
		)

		id := domain.DeploymentIDFrom(app, number)

		assert.Equal(t, app, id.AppID())
		assert.Equal(t, number, id.DeploymentNumber())
	})
}
