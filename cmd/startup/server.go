package startup

import (
	"context"
	"slices"
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
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
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
		ConnectionString() string
	}

	serverRoot struct {
		options     ServerOptions
		bus         bus.Bus
		logger      log.Logger
		db          *sqlite.Database
		usersReader domain.UsersReader
		pool        async.Pool[bus.ScheduledJob]
	}
)

// Instantiate a new server root, registering and initializing every services
// needed by the server.
func Server(options ServerOptions, logger log.Logger) (ServerRoot, error) {
	s := &serverRoot{
		options: options,
		logger:  logger,
	}

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

	// Setup auth infrastructure
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

	// Names of jobs to process in a specific group since they can take a long time
	deploymentNames := []string{
		deploy.Command{}.Name_(),
		cleanup_app.Command{}.Name_(),
	}

	s.pool = async.NewPool(
		s.logger,
		async.Poll(s.options.RunnersPollInterval(), scheduler.GetNextPendingJobs),
		async.GroupFunc(
			s.options.RunnersDeploymentCount(),
			scheduler.Process,
			func(ctx context.Context, job bus.ScheduledJob) bool {
				return slices.Contains(deploymentNames, job.Message().Name_())
			},
		),
	)

	return s, s.pool.Start()
}

func (s *serverRoot) Cleanup() error {
	s.logger.Debug("cleaning server services")

	s.pool.Stop()

	return s.db.Close()
}

func (s *serverRoot) Bus() bus.Dispatcher             { return s.bus }
func (s *serverRoot) Logger() log.Logger              { return s.logger }
func (s *serverRoot) UsersReader() domain.UsersReader { return s.usersReader }
