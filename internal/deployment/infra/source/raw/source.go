package raw

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source"
	"github.com/YuukanOO/seelf/pkg/ostools"
	"github.com/YuukanOO/seelf/pkg/types"
	"github.com/YuukanOO/seelf/pkg/validation"
	"github.com/YuukanOO/seelf/pkg/validation/strings"
)

var ErrWriteComposeFailed = errors.New("write_compose_failed")

type (
	Options interface {
		AppsDir() string
		LogsDir() string
	}

	service struct {
		options Options
	}
)

func New(options Options) source.Source {
	return &service{
		options: options,
	}
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

func (s *service) Fetch(ctx context.Context, depl domain.Deployment) error {
	logfile, err := ostools.OpenAppend(depl.LogPath(s.options.LogsDir()))

	if err != nil {
		return err
	}

	defer logfile.Close()

	logger := infra.NewStepLogger(logfile)

	buildDir := depl.Path(s.options.AppsDir())

	if err := os.RemoveAll(buildDir); err != nil {
		logger.Error(err)
		return domain.ErrSourceFetchFailed
	}

	filename := filepath.Join(buildDir, "compose.yml")

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
