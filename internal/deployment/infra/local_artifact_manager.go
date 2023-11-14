package infra

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/ostools"
)

var (
	ErrArtifactOpenLoggerFailed            = errors.New("artifact_open_logger_failed")
	ErrArtifactPrepareBuildDirectoryFailed = errors.New("artifact_prepare_build_directory_failed")
)

const (
	logsDir = "logs"
	appsDir = "apps"
)

type (
	LocalArtifactOptions interface {
		DeploymentDirTemplate() *template.Template
		DataDir() string
	}

	localArtifactManager struct {
		options       LocalArtifactOptions
		appsDirectory string
		logsDirectory string
		logger        log.Logger
	}

	deploymentTemplateData struct {
		Number      domain.DeploymentNumber
		Environment domain.Environment
	}
)

// Instantiate a new ArtifactManager which will store all the artifacts locally.
func NewLocalArtifactManager(options LocalArtifactOptions, logger log.Logger) domain.ArtifactManager {
	return &localArtifactManager{
		options:       options,
		appsDirectory: filepath.Join(options.DataDir(), appsDir),
		logsDirectory: filepath.Join(options.DataDir(), logsDir),
		logger:        logger,
	}
}

func (a *localArtifactManager) PrepareBuild(
	ctx context.Context,
	depl domain.Deployment,
) (buildDirectory string, logger domain.DeploymentLogger, err error) {
	logfile, err := ostools.OpenAppend(a.LogPath(ctx, depl))

	if err != nil {
		a.logger.Error(err)
		return "", nil, ErrArtifactOpenLoggerFailed
	}

	logger = NewStepLogger(logfile)

	defer func() {
		if err == nil {
			return
		}

		logger.Error(err)                            // Log the error in the deployment log file
		err = ErrArtifactPrepareBuildDirectoryFailed // Returns a generic error for the job to fail with it
		logger.Close()                               // And close the logger right now
	}()

	if buildDirectory, err = a.deploymentPath(depl); err != nil {
		return
	}

	logger.Infof("preparing build directory %s", buildDirectory)

	if err = ostools.EmptyDir(buildDirectory); err != nil {
		return
	}

	return
}

func (a *localArtifactManager) Cleanup(ctx context.Context, app domain.App) error {
	// Remove all app directory
	appDir := a.appPath(app.ID())
	a.logger.Debugw("removing app directory", "path", appDir)
	if err := os.RemoveAll(appDir); err != nil {
		return err
	}

	// Remove all logs for this app
	logsPattern := filepath.Join(a.logsDirectory, fmt.Sprintf("*%s*.deployment.log", app.ID()))
	a.logger.Debugw("removing app logs", "pattern", logsPattern)
	return ostools.RemovePattern(logsPattern)
}

func (a *localArtifactManager) LogPath(ctx context.Context, depl domain.Deployment) string {
	return filepath.Join(
		a.logsDirectory,
		fmt.Sprintf("%d-%s-%d.deployment.log",
			depl.Requested().At().Unix(), depl.ID().AppID(), depl.ID().DeploymentNumber()))
}

func (a *localArtifactManager) appPath(appID domain.AppID) string {
	return filepath.Join(a.appsDirectory, string(appID))
}

func (a *localArtifactManager) deploymentPath(depl domain.Deployment) (string, error) {
	var w strings.Builder

	if err := a.options.DeploymentDirTemplate().Execute(&w, deploymentTemplateData{
		Number:      depl.ID().DeploymentNumber(),
		Environment: depl.Config().Environment(),
	}); err != nil {
		return "", err
	}

	return filepath.Join(a.appPath(depl.ID().AppID()), w.String()), nil
}
