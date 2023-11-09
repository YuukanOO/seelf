package cleanup

import (
	"database/sql/driver"

	deployment "github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/worker/domain"
)

type Data deployment.AppID

func (d Data) Discriminator() string        { return "deployment.cleanup-app" }
func (d Data) Value() (driver.Value, error) { return string(d), nil }

func init() {
	domain.JobDataTypes.Register(Data(""), func(value string) (domain.JobData, error) {
		return Data(value), nil
	})
}
