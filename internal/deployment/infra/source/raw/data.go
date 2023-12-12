package raw

import (
	"github.com/YuukanOO/seelf/internal/deployment/app/get_deployment"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
)

type Data string

func (p Data) Discriminator() string { return "raw" }
func (p Data) NeedVCS() bool         { return false }

func init() {
	domain.SourceDataTypes.Register(Data(""), func(value string) (domain.SourceData, error) {
		return Data(value), nil
	})

	get_deployment.SourceDataTypes.Register(Data(""), func(value string) (get_deployment.SourceData, error) {
		return Data(value), nil
	})
}
