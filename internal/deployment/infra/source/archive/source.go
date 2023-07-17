package archive

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/ostools"
)

const (
	kind       domain.Kind = "archive"
	tmpPattern string      = "seelf-archive-"
)

var (
	ErrOpenArchiveFailed    = errors.New("open_archive_failed")
	ErrUnzipArchiveFailed   = errors.New("unzip_archive_failed")
	ErrReadArchiveFailed    = errors.New("read_archive_failed")
	ErrWriteFileFailed      = errors.New("write_archive_file_failed")
	ErrWriteDirectoryFailed = errors.New("write_archive_directory_failed")
)

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
	_, ok := payload.(*multipart.FileHeader)
	return ok
}

func (t *service) Prepare(app domain.App, payload any) (domain.Meta, error) {
	file, ok := payload.(*multipart.FileHeader)

	if !ok {
		return domain.Meta{}, domain.ErrInvalidSourcePayload
	}

	tmpfile, err := os.CreateTemp("", tmpPattern)

	if err != nil {
		return domain.Meta{}, err
	}

	defer tmpfile.Close()

	archive, err := file.Open()

	if err != nil {
		return domain.Meta{}, err
	}

	defer archive.Close()

	if _, err := io.Copy(tmpfile, archive); err != nil {
		return domain.Meta{}, err
	}

	return domain.NewMeta(kind, tmpfile.Name()), nil
}

func (*service) CanFetch(meta domain.Meta) bool {
	return meta.Kind() == kind
}

func (t *service) Fetch(ctx context.Context, depl domain.Deployment) error {
	logfile, err := ostools.OpenAppend(depl.LogPath(t.options.LogsDir()))

	if err != nil {
		return err
	}

	defer logfile.Close()

	logger := log.NewStepLogger(logfile)

	buildDir := depl.Path(t.options.AppsDir())

	if err := ostools.EmptyDir(buildDir); err != nil {
		logger.Error(err)
		return domain.ErrSourceFetchFailed
	}

	archivePath := depl.Source().Data()

	logger.Stepf("extracting archive %s into %s", archivePath, buildDir)

	// Open the archive file stored in a temporary location
	archive, err := os.Open(archivePath)

	if err != nil {
		logger.Error(err)
		return ErrOpenArchiveFailed
	}

	defer archive.Close()

	// And now, uncompress and untar it in the app directory
	gzr, err := gzip.NewReader(archive)

	if err != nil {
		logger.Error(err)
		return ErrUnzipArchiveFailed
	}

	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			logger.Error(err)
			return ErrReadArchiveFailed

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(buildDir, header.Name)

		// check the file type
		switch header.Typeflag {
		case tar.TypeDir:
			if err := ostools.MkdirAll(target); err != nil {
				logger.Error(err)
				return ErrWriteDirectoryFailed
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))

			if err != nil {
				logger.Error(err)
				return ErrWriteFileFailed
			}

			if _, err := io.Copy(f, tr); err != nil {
				logger.Error(err)
				return ErrWriteFileFailed
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}
}
