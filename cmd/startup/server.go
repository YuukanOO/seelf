package startup

import (
	"context"

	"github.com/YuukanOO/seelf/internal/auth/app/create_first_account"
	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	authinfra "github.com/YuukanOO/seelf/internal/auth/infra"
	"github.com/YuukanOO/seelf/internal/deployment/app/expose_seelf_container"
	deployment "github.com/YuukanOO/seelf/internal/deployment/domain"
	deploymentinfra "github.com/YuukanOO/seelf/internal/deployment/infra"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/embedded"
	bussqlite "github.com/YuukanOO/seelf/pkg/bus/sqlite"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
)

type (
	ServerOptions interface {
		deploymentinfra.Options

		AppExposedUrl() monad.Maybe[deployment.Url]
		DefaultEmail() string
		DefaultPassword() string
		ConnectionString() string
		RunnersDefinitions() ([]embedded.RunnerDefinition, error)
	}

	ServerRoot struct {
		bus          bus.Bus
		logger       log.Logger
		db           *sqlite.Database
		orchestrator *embedded.Orchestrator
	}
)

// Instantiate a new server root, registering and initializing every services
// needed by the server.
func Server(options ServerOptions, logger log.Logger) (root *ServerRoot, err error) {
	s := &ServerRoot{
		logger: logger,
	}

	s.bus = embedded.NewBus()

	db, err := sqlite.Open(options.ConnectionString(), s.logger, s.bus)

	if err != nil {
		return
	}

	// Close the database on error
	defer func() {
		if err == nil {
			return
		}

		_ = db.Close()
	}()

	s.db = db

	jobsStore, err := bussqlite.Setup(s.bus, s.db)

	if err != nil {
		return
	}

	// Setup auth infrastructure
	if err = authinfra.Setup(s.logger, s.db, s.bus); err != nil {
		return
	}

	// Setups deployment infrastructure
	if err = deploymentinfra.Setup(
		options,
		s.logger,
		s.db,
		s.bus,
		jobsStore,
	); err != nil {
		return
	}

	// Build the background jobs orchestrator
	runners, err := options.RunnersDefinitions()

	if err != nil {
		return
	}

	s.orchestrator = embedded.NewOrchestrator(jobsStore, s.bus, s.logger, runners...)

	// Create the first account if needed
	uid, err := bus.Send(s.bus, context.Background(), create_first_account.Command{
		Email:    options.DefaultEmail(),
		Password: options.DefaultPassword(),
	})

	if err != nil {
		return
	}

	// Create the target needed to expose seelf itself and manage certificates if needed
	if exposedUrl, isSet := options.AppExposedUrl().TryGet(); isSet {
		container := exposedUrl.User().Get("")

		s.logger.Infow("exposing seelf container using the local target, creating it if needed, the container may restart once done",
			"container", container)

		if _, err = bus.Send(s.bus, auth.WithUserID(context.Background(), auth.UserID(uid)), expose_seelf_container.Command{
			Container: container,
			Url:       exposedUrl.WithoutUser().String(),
		}); err != nil {
			return
		}
	}

	s.orchestrator.Start()
	root = s
	return
}

func (s *ServerRoot) Cleanup() error {
	s.logger.Debug("cleaning server services")

	s.orchestrator.Stop()

	return s.db.Close()
}

func (s *ServerRoot) Bus() bus.Dispatcher { return s.bus }
func (s *ServerRoot) Logger() log.Logger  { return s.logger }
