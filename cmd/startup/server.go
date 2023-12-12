package startup

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/YuukanOO/seelf/internal/auth/app/create_first_account"
	"github.com/YuukanOO/seelf/internal/auth/app/login"
	"github.com/YuukanOO/seelf/internal/auth/app/update_user"
	authdomain "github.com/YuukanOO/seelf/internal/auth/domain"
	authinfra "github.com/YuukanOO/seelf/internal/auth/infra"
	authsqlite "github.com/YuukanOO/seelf/internal/auth/infra/sqlite"
	"github.com/YuukanOO/seelf/internal/deployment/app/cleanup_app"
	"github.com/YuukanOO/seelf/internal/deployment/app/create_app"
	"github.com/YuukanOO/seelf/internal/deployment/app/deploy"
	"github.com/YuukanOO/seelf/internal/deployment/app/fail_running_deployments"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_deployment_log"
	"github.com/YuukanOO/seelf/internal/deployment/app/queue_deployment"
	"github.com/YuukanOO/seelf/internal/deployment/app/update_app"
	deploymentdomain "github.com/YuukanOO/seelf/internal/deployment/domain"
	deploymentinfra "github.com/YuukanOO/seelf/internal/deployment/infra"
	"github.com/YuukanOO/seelf/internal/deployment/infra/backend/docker"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/archive"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/git"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/raw"
	deploymentsqlite "github.com/YuukanOO/seelf/internal/deployment/infra/sqlite"
	"github.com/YuukanOO/seelf/internal/worker/app/fail_running_jobs"
	"github.com/YuukanOO/seelf/internal/worker/app/process"
	"github.com/YuukanOO/seelf/internal/worker/app/queue"
	workerdomain "github.com/YuukanOO/seelf/internal/worker/domain"
	"github.com/YuukanOO/seelf/internal/worker/infra/jobs"
	"github.com/YuukanOO/seelf/internal/worker/infra/jobs/cleanup"
	deployjob "github.com/YuukanOO/seelf/internal/worker/infra/jobs/deploy"
	workersqlite "github.com/YuukanOO/seelf/internal/worker/infra/sqlite"
	"github.com/YuukanOO/seelf/pkg/async"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/ostools"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
)

var (
	errServerReset                         = errors.New("server_reset")
	configFingerprintName                  = "last_run_data"
	noticeNotSupportedConfigChangeDetected = `looks like you have changed the domain used by seelf for your apps (either the protocol or the domain itself).
	
	Those changes are not supported yet. For now, for things to keep running correctly, you'll have to manually redeploy all of your apps.`
	noticeSecretKeyGenerated = `a default secret key has been generated. If you want to override it, you can set the HTTP_SECRET environment variable.`
)

type (
	// Represents a services root containing every services used by a server.
	ServerRoot interface {
		Cleanup() error
		Bus() bus.Bus
		Logger() log.Logger
		UsersReader() authdomain.UsersReader
	}

	ServerOptions interface {
		docker.Options
		deploymentinfra.LocalArtifactOptions

		RunnersPollInterval() time.Duration
		RunnersDeploymentCount() int
		IsVerbose() bool
		ConnectionString() string
		DefaultEmail() string
		DefaultPassword() string
		IsUsingGeneratedSecret() bool
		ConfigPath() string
	}

	serverRoot struct {
		options     ServerOptions
		bus         bus.Bus
		logger      log.Logger
		db          sqlite.Database
		usersReader authdomain.UsersReader
		docker      docker.Backend
		pool        async.Pool
	}
)

// Instantiate a new server root, registering and initializing every services
// needed by the server.
func Server(options ServerOptions) (ServerRoot, error) {
	s := &serverRoot{options: options}

	if err := s.configureServices(); err != nil {
		return nil, err
	}

	if err := s.start(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *serverRoot) Cleanup() error {
	s.logger.Debug("cleaning server services")

	s.pool.Stop()

	return s.db.Close()
}

func (s *serverRoot) Bus() bus.Bus                        { return s.bus }
func (s *serverRoot) Logger() log.Logger                  { return s.logger }
func (s *serverRoot) UsersReader() authdomain.UsersReader { return s.usersReader }

func (s *serverRoot) configureServices() error {
	s.logger = log.NewLogger(s.options.IsVerbose())
	s.logger.Infow("configuration loaded", "path", s.options.ConfigPath())

	s.bus = bus.NewInMemoryBus()

	db, err := sqlite.Open(s.options.ConnectionString(), s.logger, s.bus)

	if err != nil {
		return err
	}

	s.db = db

	handler := jobs.NewFacade(
		deployjob.New(s.logger, s.bus),
		cleanup.New(s.logger, s.bus),
	)

	usersStore := authsqlite.NewUsersStore(s.db)
	authQueryHandler := authsqlite.NewGateway(s.db)
	appsStore := deploymentsqlite.NewAppsStore(s.db)
	deploymentsStore := deploymentsqlite.NewDeploymentsStore(s.db)
	deploymentQueryHandler := deploymentsqlite.NewGateway(s.db)
	jobsStore := workersqlite.NewJobsStore(s.db)

	passwordHasher := authinfra.NewBCryptHasher()
	keyGenerator := authinfra.NewKeyGenerator()

	s.docker = docker.New(s.options, s.logger)
	artifactManager := deploymentinfra.NewLocalArtifactManager(s.options, s.logger)

	sourceFacade := source.NewFacade(
		raw.New(),
		archive.New(),
		git.New(appsStore),
	)

	// Auth
	bus.Register(s.bus, login.Handler(usersStore, passwordHasher))
	bus.Register(s.bus, create_first_account.Handler(usersStore, usersStore, passwordHasher, keyGenerator))
	bus.Register(s.bus, update_user.Handler(usersStore, usersStore, passwordHasher))
	bus.Register(s.bus, authQueryHandler.GetProfile)

	// Deployment
	bus.Register(s.bus, create_app.Handler(appsStore, appsStore))
	bus.Register(s.bus, update_app.Handler(appsStore, appsStore))
	bus.Register(s.bus, queue_deployment.Handler(appsStore, deploymentsStore, deploymentsStore, sourceFacade))
	bus.Register(s.bus, deploy.Handler(deploymentsStore, deploymentsStore, artifactManager, sourceFacade, s.docker))
	bus.Register(s.bus, fail_running_deployments.Handler(deploymentsStore, deploymentsStore))
	bus.Register(s.bus, cleanup_app.Handler(deploymentsStore, appsStore, appsStore, artifactManager, s.docker))
	bus.Register(s.bus, get_deployment_log.Handler(deploymentsStore, artifactManager))
	bus.Register(s.bus, deploymentQueryHandler.GetAllApps)
	bus.Register(s.bus, deploymentQueryHandler.GetAppByID)
	bus.Register(s.bus, deploymentQueryHandler.GetAllDeploymentsByApp)
	bus.Register(s.bus, deploymentQueryHandler.GetDeploymentByID)

	// TODO: since worker jobs are just dispatch of a predefined command in another domain, maybe
	// we can leverage this to make it more generic.

	bus.On(s.bus, func(ctx context.Context, evt deploymentdomain.DeploymentCreated) error {
		_, err := bus.Send(s.bus, ctx, queue.Command{Payload: deployjob.Request(evt)})
		return err
	})

	bus.On(s.bus, func(ctx context.Context, evt deploymentdomain.AppCleanupRequested) error {
		_, err := bus.Send(s.bus, ctx, queue.Command{Payload: cleanup.Request(evt)})
		return err
	})

	// Worker
	bus.Register(s.bus, fail_running_jobs.Handler(jobsStore, jobsStore))
	bus.Register(s.bus, queue.Handler(jobsStore, handler))

	s.pool = async.NewPool(
		s.logger,
		s.options.RunnersPollInterval(),
		jobsStore.GetNextPendingJobs,
		func(job workerdomain.Job) string { return job.Data().Discriminator() },
		async.Group(
			s.options.RunnersDeploymentCount(),
			process.Handler(jobsStore, handler),
			deployjob.Data{}.Discriminator(), cleanup.Data("").Discriminator()),
	)

	return nil
}

func (s *serverRoot) start() error {
	if s.options.IsUsingGeneratedSecret() {
		s.logger.Info(noticeSecretKeyGenerated)
	}

	// Migrate the database first
	if err := s.db.Migrate(
		authsqlite.Migrations,
		deploymentsqlite.Migrations,
		workersqlite.Migrations,
	); err != nil {
		return err
	}

	// Checks for unsupported configuration modifications
	if err := s.checkNonSupportedConfigChanges(); err != nil {
		return err
	}

	ctx := context.Background()

	// Fail stale deployments
	if _, err := bus.Send(s.bus, ctx, fail_running_deployments.Command{
		Reason: errServerReset,
	}); err != nil {
		return err
	}

	// Fail stale jobs
	if _, err := bus.Send(s.bus, ctx, fail_running_jobs.Command{
		Reason: errServerReset,
	}); err != nil {
		return err
	}

	if _, err := bus.Send(s.bus, ctx, create_first_account.Command{
		Email:    s.options.DefaultEmail(),
		Password: s.options.DefaultPassword(),
	}); err != nil {
		return err
	}

	if err := s.docker.Setup(); err != nil {
		return err
	}

	go s.pool.Start()

	return nil
}

func (s *serverRoot) checkNonSupportedConfigChanges() error {
	fingerprintPath := filepath.Join(s.options.DataDir(), configFingerprintName)
	fingerprint := s.options.Domain().String()

	data, err := os.ReadFile(fingerprintPath)

	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	strdata := string(data)

	if strdata != "" && strdata != fingerprint {
		s.logger.Warn(noticeNotSupportedConfigChangeDetected)
	}

	return ostools.WriteFile(fingerprintPath, []byte(fingerprint))
}
