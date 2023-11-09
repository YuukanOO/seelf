package git

import (
	"database/sql/driver"
	"encoding/json"
	"strings"

	"github.com/YuukanOO/seelf/internal/deployment/app/query"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/storage"
)

type Data struct {
	Branch string `json:"branch"`
	Hash   string `json:"hash"`
}

func (p Data) Discriminator() string { return "git" }
func (p Data) NeedVCS() bool         { return true }

func (p Data) Value() (driver.Value, error) { return storage.ValueJSON(p) }

func init() {
	domain.SourceDataTypes.Register(Data{}, func(value string) (domain.SourceData, error) {
		return tryParseGitData(value)
	})

	// Here the registered discriminated type is the same since there are no unexposed fields and
	// it also handle the retrocompatibility with the old payload format.
	query.SourceDataTypes.Register(Data{}, func(value string) (query.SourceData, error) {
		return tryParseGitData(value)
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
