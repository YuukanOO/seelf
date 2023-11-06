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
	"github.com/YuukanOO/seelf/pkg/validation"
	"github.com/YuukanOO/seelf/pkg/validation/strings"
)

const kind domain.Kind = "raw"

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

func (*service) CanPrepare(payload any) bool {
	_, ok := payload.(string)
	return ok
}

func (s *service) Prepare(app domain.App, payload any) (domain.Meta, error) {
	rawServiceFileContent, ok := payload.(string)

	if !ok {
		return domain.Meta{}, domain.ErrInvalidSourcePayload
	}

	if err := validation.Check(validation.Of{
		"content": validation.Is(rawServiceFileContent, strings.Required),
	}); err != nil {
		return domain.Meta{}, err
	}

	return domain.NewMeta(kind, rawServiceFileContent), nil
}

func (*service) CanFetch(meta domain.Meta) bool {
	return meta.Kind() == kind
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

	logger.Stepf("writing service file to %s", filename)

	if err := ostools.WriteFile(filename, []byte(depl.Source().Data())); err != nil {
		logger.Error(err)
		return ErrWriteComposeFailed
	}

	return nil
}
