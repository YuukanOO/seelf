package deploy

import (
	"database/sql/driver"
	"encoding/json"
	"strconv"
	"strings"

	deployment "github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/pkg/storage"
)

type Data struct {
	AppID            deployment.AppID            `json:"app_id"`
	DeploymentNumber deployment.DeploymentNumber `json:"deployment_number"`
}

func (d Data) Discriminator() string        { return "deployment.deploy" }
func (d Data) Value() (driver.Value, error) { return storage.ValueJSON(d) }

func init() {
	domain.JobDataTypes.Register(Data{}, func(value string) (domain.JobData, error) {
		return tryParseDeployJobData(value)
	})
}

func tryParseDeployJobData(value string) (domain.JobData, error) {
	var data Data

	// Handle old payload for compatibility.
	if !json.Valid([]byte(value)) {
		separatorIdx := strings.Index(value, ":")
		appid, numberStr := value[:separatorIdx], value[separatorIdx+1:]
		number, err := strconv.Atoi(numberStr)

		if err != nil {
			return nil, err
		}

		data.AppID = deployment.AppID(appid)
		data.DeploymentNumber = deployment.DeploymentNumber(number)

		return data, nil
	}

	err := storage.ScanJSON(value, &data)

	return data, err
}
