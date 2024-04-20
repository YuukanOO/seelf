package startup

import (
	"context"
	"time"

	"github.com/YuukanOO/seelf/internal/auth/app/create_first_account"
	"github.com/YuukanOO/seelf/internal/auth/domain"
	authinfra "github.com/YuukanOO/seelf/internal/auth/infra"
	"github.com/YuukanOO/seelf/internal/deployment/app/cleanup_app"
	"github.com/YuukanOO/seelf/internal/deployment/app/cleanup_target"
	"github.com/YuukanOO/seelf/internal/deployment/app/configure_target"
	"github.com/YuukanOO/seelf/internal/deployment/app/delete_app"
	"github.com/YuukanOO/seelf/internal/deployment/app/delete_target"
	"github.com/YuukanOO/seelf/internal/deployment/app/deploy"
	"github.com/YuukanOO/seelf/internal/deployment/app/expose_seelf_container"
	deploymentdomain "github.com/YuukanOO/seelf/internal/deployment/domain"
	deploymentinfra "github.com/YuukanOO/seelf/internal/deployment/infra"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/memory"
	bussqlite "github.com/YuukanOO/seelf/pkg/bus/sqlite"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
)

type (
	// Represents a services root containing every services used by a server.
	ServerRoot interface {
		Cleanup() error
		Bus() bus.Dispatcher
		Logger() log.Logger
		UsersReader() domain.UsersReader
		ScheduledJobsStore() bus.ScheduledJobsStore
	}

	ServerOptions interface {
		deploymentinfra.Options

		AppExposedUrl() monad.Maybe[deploymentdomain.Url]
		DefaultEmail() string
		DefaultPassword() string
		RunnersPollInterval() time.Duration
		RunnersDeploymentCount() int
		RunnersCleanupCount() int
		ConnectionString() string
	}

	serverRoot struct {
		options        ServerOptions
		bus            bus.Bus
		logger         log.Logger
		db             *sqlite.Database
		usersReader    domain.UsersReader
		schedulerStore bus.ScheduledJobsStore
		scheduler      bus.RunnableScheduler
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

	s.schedulerStore = bussqlite.NewScheduledJobsStore(s.db)

	if err = s.schedulerStore.Setup(); err != nil {
		return nil, err
	}

	s.scheduler = bus.NewScheduler(s.schedulerStore, s.logger, s.bus, s.options.RunnersPollInterval(),
		bus.WorkerGroup{
			Size:     s.options.RunnersDeploymentCount(),
			Messages: []string{deploy.Command{}.Name_()},
		},
		bus.WorkerGroup{
			Size: s.options.RunnersCleanupCount(),
			Messages: []string{
				cleanup_app.Command{}.Name_(),
				delete_app.Command{}.Name_(),
				configure_target.Command{}.Name_(),
				cleanup_target.Command{}.Name_(),
				delete_target.Command{}.Name_(),
			},
		},
	)

	// Setup auth infrastructure
	if s.usersReader, err = authinfra.Setup(s.logger, s.db, s.bus); err != nil {
		return nil, err
	}

	// Setups deployment infrastructure
	if err = deploymentinfra.Setup(
		s.options,
		s.logger,
		s.db,
		s.bus,
		s.scheduler,
	); err != nil {
		return nil, err
	}

	// Create the first account if needed
	uid, err := bus.Send(s.bus, context.Background(), create_first_account.Command{
		Email:    options.DefaultEmail(),
		Password: options.DefaultPassword(),
	})

	if err != nil {
		return nil, err
	}

	// Create the target needed to expose seelf itself and manage certificates if needed
	if exposedUrl, isSet := s.options.AppExposedUrl().TryGet(); isSet {
		container := exposedUrl.User().Get("")

		s.logger.Infow("exposing seelf container using the local target, creating it if needed, the container may restart once done",
			"container", container)

		if _, err := bus.Send(s.bus, domain.WithUserID(context.Background(), domain.UserID(uid)), expose_seelf_container.Command{
			Container: container,
			Url:       exposedUrl.WithoutUser().String(),
		}); err != nil {
			return nil, err
		}
	}

	s.scheduler.Start()

	return s, nil
}

func (s *serverRoot) Cleanup() error {
	s.logger.Debug("cleaning server services")

	s.scheduler.Stop()

	return s.db.Close()
}

func (s *serverRoot) Bus() bus.Dispatcher                        { return s.bus }
func (s *serverRoot) Logger() log.Logger                         { return s.logger }
func (s *serverRoot) UsersReader() domain.UsersReader            { return s.usersReader }
func (s *serverRoot) ScheduledJobsStore() bus.ScheduledJobsStore { return s.schedulerStore }
