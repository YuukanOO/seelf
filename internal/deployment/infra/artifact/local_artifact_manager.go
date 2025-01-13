package artifact

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
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
	LocalOptions interface {
		DeploymentDirTemplate() *template.Template
		DataDir() string
	}

	localArtifactManager struct {
		options       LocalOptions
		appsDirectory string
		logsDirectory string
		logger        log.Logger
	}

	deploymentTemplateData struct {
		Number      domain.DeploymentNumber
		Environment domain.EnvironmentName
	}
)

// Instantiate a new ArtifactManager which will store all the artifacts locally.
func NewLocal(options LocalOptions, logger log.Logger) domain.ArtifactManager {
	return &localArtifactManager{
		options:       options,
		appsDirectory: filepath.Join(options.DataDir(), appsDir),
		logsDirectory: filepath.Join(options.DataDir(), logsDir),
		logger:        logger,
	}
}

func (a *localArtifactManager) PrepareBuild(
	ctx context.Context,
	deployment domain.Deployment,
) (domain.DeploymentContext, error) {
	logFile, err := ostools.OpenAppend(a.LogPath(ctx, deployment))

	if err != nil {
		a.logger.Error(err)
		return domain.DeploymentContext{}, ErrArtifactOpenLoggerFailed
	}

	logger := newLogger(logFile)

	defer func() {
		if err == nil {
			return
		}

		logger.Error(err)                            // Log the error in the deployment log file
		err = ErrArtifactPrepareBuildDirectoryFailed // Returns a generic error for the job to fail with it
		logger.Close()                               // And close the logger right now
	}()

	buildDirectory, err := a.deploymentPath(deployment)

	if err != nil {
		return domain.DeploymentContext{}, err
	}

	logger.Infof("preparing build directory %s", buildDirectory)

	if err = ostools.EmptyDir(buildDirectory); err != nil {
		return domain.DeploymentContext{}, err
	}

	return domain.NewDeploymentContext(buildDirectory, logger), nil
}

func (a *localArtifactManager) Cleanup(ctx context.Context, id domain.AppID) error {
	// Remove all app directory
	appDir := a.appPath(id)
	a.logger.Debugw("removing app directory", "path", appDir)
	if err := os.RemoveAll(appDir); err != nil {
		return err
	}

	// Remove all logs for this app
	logsPattern := filepath.Join(a.logsDirectory, "*"+string(id)+"*.deployment.log")
	a.logger.Debugw("removing app logs", "pattern", logsPattern)
	return ostools.RemovePattern(logsPattern)
}

func (a *localArtifactManager) LogPath(ctx context.Context, depl domain.Deployment) string {
	return filepath.Join(
		a.logsDirectory,
		strconv.FormatInt(depl.Requested().At().Unix(), 10)+
			"-"+
			string(depl.ID().AppID())+
			"-"+
			strconv.Itoa(int(depl.ID().DeploymentNumber()))+
			".deployment.log",
	)
}

func (a *localArtifactManager) appPath(appID domain.AppID) string {
	return filepath.Join(a.appsDirectory, string(appID))
}

func (a *localArtifactManager) deploymentPath(deployment domain.Deployment) (string, error) {
	var w strings.Builder

	if err := a.options.DeploymentDirTemplate().Execute(&w, deploymentTemplateData{
		Number:      deployment.ID().DeploymentNumber(),
		Environment: deployment.Config().Environment(),
	}); err != nil {
		return "", err
	}

	return filepath.Join(a.appPath(deployment.ID().AppID()), w.String()), nil
}
