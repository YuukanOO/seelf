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
	"github.com/YuukanOO/seelf/pkg/ostools"
	"github.com/YuukanOO/seelf/pkg/types"
)

const tmpPattern string = "seelf-archive-"

var (
	ErrOpenArchiveFailed    = errors.New("open_archive_failed")
	ErrUnzipArchiveFailed   = errors.New("unzip_archive_failed")
	ErrReadArchiveFailed    = errors.New("read_archive_failed")
	ErrWriteFileFailed      = errors.New("write_archive_file_failed")
	ErrWriteDirectoryFailed = errors.New("write_archive_directory_failed")
)

type service struct{}

func New() source.Source {
	return &service{}
}

func (*service) CanPrepare(payload any) bool          { return types.Is[*multipart.FileHeader](payload) }
func (*service) CanFetch(meta domain.SourceData) bool { return types.Is[Data](meta) }

func (t *service) Prepare(app domain.App, payload any) (domain.SourceData, error) {
	file, ok := payload.(*multipart.FileHeader)

	if !ok {
		return nil, domain.ErrInvalidSourcePayload
	}

	tmpfile, err := os.CreateTemp("", tmpPattern)

	if err != nil {
		return nil, err
	}

	defer tmpfile.Close()

	archive, err := file.Open()

	if err != nil {
		return nil, err
	}

	defer archive.Close()

	if _, err := io.Copy(tmpfile, archive); err != nil {
		return nil, err
	}

	return Data(tmpfile.Name()), nil
}

func (t *service) Fetch(ctx context.Context, dir string, logger domain.DeploymentLogger, depl domain.Deployment) error {
	data, ok := depl.Source().(Data)

	if !ok {
		return domain.ErrInvalidSourcePayload
	}

	logger.Stepf("extracting archive %s into %s", data, dir)

	// Open the archive file stored in a temporary location
	archive, err := os.Open(string(data))

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
		target := filepath.Join(dir, header.Name)

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
