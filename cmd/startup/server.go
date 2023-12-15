package startup

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	authinfra "github.com/YuukanOO/seelf/internal/auth/infra"
	"github.com/YuukanOO/seelf/internal/deployment/app/cleanup_app"
	"github.com/YuukanOO/seelf/internal/deployment/app/deploy"
	deploymentinfra "github.com/YuukanOO/seelf/internal/deployment/infra"
	"github.com/YuukanOO/seelf/pkg/async"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/memory"
	bussqlite "github.com/YuukanOO/seelf/pkg/bus/sqlite"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/ostools"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
)

var (
	configFingerprintName                  = "last_run_data"
	noticeNotSupportedConfigChangeDetected = `looks like you have changed the domain used by seelf for your apps (either the protocol or the domain itself).
	
	Those changes are not supported yet. For now, for things to keep running correctly, you'll have to manually redeploy all of your apps.`
	noticeSecretKeyGenerated = `a default secret key has been generated. If you want to override it, you can set the HTTP_SECRET environment variable.`
)

type (
	// Represents a services root containing every services used by a server.
	ServerRoot interface {
		Cleanup() error
		Bus() bus.Dispatcher
		Logger() log.Logger
		UsersReader() domain.UsersReader
	}

	ServerOptions interface {
		deploymentinfra.Options
		authinfra.Options

		RunnersPollInterval() time.Duration
		RunnersDeploymentCount() int
		IsVerbose() bool
		ConnectionString() string
		IsUsingGeneratedSecret() bool
		ConfigPath() string
	}

	serverRoot struct {
		options     ServerOptions
		bus         bus.Bus
		logger      log.Logger
		db          *sqlite.Database
		usersReader domain.UsersReader
		pool        async.Pool
	}
)

// Instantiate a new server root, registering and initializing every services
// needed by the server.
func Server(options ServerOptions) (ServerRoot, error) {
	s := &serverRoot{options: options}

	s.logger = log.NewLogger(s.options.IsVerbose())
	s.logger.Infow("configuration loaded",
		"path", s.options.ConfigPath())

	s.bus = memory.NewBus()

	db, err := sqlite.Open(s.options.ConnectionString(), s.logger, s.bus)

	if err != nil {
		return nil, err
	}

	s.db = db

	adapter := bussqlite.NewSchedulerAdapter(s.db)

	if err = adapter.Setup(); err != nil {
		return nil, err
	}

	scheduler := bus.NewScheduler(adapter, s.logger, s.bus)

	// Setups auth infrastructure
	s.usersReader, err = authinfra.Setup(s.options, s.logger, s.db, s.bus)

	if err != nil {
		return nil, err
	}

	// Setups deployment infrastructure
	if err = deploymentinfra.Setup(
		s.options,
		s.logger,
		s.db,
		s.bus,
		scheduler,
	); err != nil {
		return nil, err
	}

	if s.options.IsUsingGeneratedSecret() {
		s.logger.Info(noticeSecretKeyGenerated)
	}

	// Checks for unsupported configuration modifications
	if err := s.checkNonSupportedConfigChanges(); err != nil {
		return nil, err
	}

	s.pool = async.NewPool(
		s.logger,
		s.options.RunnersPollInterval(),
		scheduler.GetNextPendingJobs,
		func(job bus.ScheduledJob) string { return job.Message().Name_() },
		async.Group(
			s.options.RunnersDeploymentCount(),
			scheduler.Process,
			deploy.Command{}.Name_(), cleanup_app.Command{}.Name_()),
	)

	go s.pool.Start()

	return s, nil
}

func (s *serverRoot) Cleanup() error {
	s.logger.Debug("cleaning server services")

	s.pool.Stop()

	return s.db.Close()
}

func (s *serverRoot) Bus() bus.Dispatcher             { return s.bus }
func (s *serverRoot) Logger() log.Logger              { return s.logger }
func (s *serverRoot) UsersReader() domain.UsersReader { return s.usersReader }

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
