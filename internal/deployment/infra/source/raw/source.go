package raw

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source"
	"github.com/YuukanOO/seelf/pkg/ostools"
	"github.com/YuukanOO/seelf/pkg/types"
	"github.com/YuukanOO/seelf/pkg/validation"
	"github.com/YuukanOO/seelf/pkg/validation/strings"
)

var ErrWriteComposeFailed = errors.New("write_compose_failed")

type service struct{}

func New() source.Source {
	return &service{}
}

func (*service) CanPrepare(payload any) bool          { return types.Is[string](payload) }
func (*service) CanFetch(meta domain.SourceData) bool { return types.Is[Data](meta) }

func (s *service) Prepare(app domain.App, payload any) (domain.SourceData, error) {
	rawServiceFileContent, ok := payload.(string)

	if !ok {
		return nil, domain.ErrInvalidSourcePayload
	}

	if err := validation.Check(validation.Of{
		"content": validation.Is(rawServiceFileContent, strings.Required),
	}); err != nil {
		return nil, err
	}

	return Data(rawServiceFileContent), nil
}

func (s *service) Fetch(ctx context.Context, dir string, logger domain.DeploymentLogger, depl domain.Deployment) error {
	filename := filepath.Join(dir, "compose.yml")

	data, ok := depl.Source().(Data)

	if !ok {
		return domain.ErrInvalidSourcePayload
	}

	logger.Stepf("writing service file to %s", filename)

	if err := ostools.WriteFile(filename, []byte(data)); err != nil {
		logger.Error(err)
		return ErrWriteComposeFailed
	}

	return nil
}
