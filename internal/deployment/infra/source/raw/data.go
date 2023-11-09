package raw

import (
	"github.com/YuukanOO/seelf/internal/deployment/app/query"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
)

type Data string

func (p Data) Discriminator() string { return "raw" }
func (p Data) NeedVCS() bool         { return false }

func init() {
	domain.SourceDataTypes.Register(Data(""), func(value string) (domain.SourceData, error) {
		return Data(value), nil
	})

	query.SourceDataTypes.Register(Data(""), func(value string) (query.SourceData, error) {
		return Data(value), nil
	})
}
