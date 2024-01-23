package git

import (
	"database/sql/driver"
	"encoding/json"
	"strings"

	"github.com/YuukanOO/seelf/internal/deployment/app/get_deployment"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/storage"
)

type Data struct {
	Branch string `json:"branch"`
	Hash   string `json:"hash"`
}

func (p Data) Kind() string                 { return "git" }
func (p Data) NeedVersionControl() bool     { return true }
func (p Data) Value() (driver.Value, error) { return storage.ValueJSON(p) }

func init() {
	domain.SourceDataTypes.Register(Data{}, func(s string) (domain.SourceData, error) {
		return tryParseGitData(s)
	})

	// Here the registered discriminated type is the same since there are no unexposed fields and
	// it also handle the retrocompatibility with the old payload format.
	get_deployment.SourceDataTypes.Register(Data{}, func(s string) (get_deployment.SourceData, error) {
		return tryParseGitData(s)
	})
}

// Try to parse the given value as a git data payload. If the value is not a valid
// json string, it will fallback to the old format.
func tryParseGitData(value string) (Data, error) {
	var p Data

	if !json.Valid([]byte(value)) {
		lastSeparatorIdx := strings.LastIndex(value, "@")
		branch, hash := value[:lastSeparatorIdx], value[lastSeparatorIdx+1:]

		p.Branch = branch
		p.Hash = hash
		return p, nil
	}

	err := storage.ScanJSON(value, &p)

	return p, err
}
