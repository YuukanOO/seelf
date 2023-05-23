package serve

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	authcmd "github.com/YuukanOO/seelf/internal/auth/app/command"
	authquery "github.com/YuukanOO/seelf/internal/auth/app/query"
	authdomain "github.com/YuukanOO/seelf/internal/auth/domain"
	authinfra "github.com/YuukanOO/seelf/internal/auth/infra"
	authsqlite "github.com/YuukanOO/seelf/internal/auth/infra/sqlite"
	deplcmd "github.com/YuukanOO/seelf/internal/deployment/app/command"
	deplquery "github.com/YuukanOO/seelf/internal/deployment/app/query"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/backend/docker"
	deplsqlite "github.com/YuukanOO/seelf/internal/deployment/infra/sqlite"
	"github.com/YuukanOO/seelf/internal/deployment/infra/trigger"
	"github.com/YuukanOO/seelf/internal/deployment/infra/trigger/archive"
	"github.com/YuukanOO/seelf/internal/deployment/infra/trigger/git"
	"github.com/YuukanOO/seelf/internal/deployment/infra/trigger/raw"
	workercmd "github.com/YuukanOO/seelf/internal/worker/app/command"
	workerinfra "github.com/YuukanOO/seelf/internal/worker/infra"
	"github.com/YuukanOO/seelf/internal/worker/infra/jobs"
	workersqlite "github.com/YuukanOO/seelf/internal/worker/infra/sqlite"
	"github.com/YuukanOO/seelf/pkg/async"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/ostools"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
	"github.com/gin-contrib/pprof"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

//go:embed all:front/build/*
var front embed.FS

const (
	embeddedRootDir = "front/build"
	sessionName     = "seelf"

	configFingerprintName                  = "last_run_data"
	noticeNotSupportedConfigChangeDetected = `looks like you have changed the domain used by seelf for your apps (either the protocol or the domain itself).
	
	Those changes are not supported yet. For now, for things to keep running correctly, you'll have to manually redeploy all of your apps.`
	noticeSecretKeyGenerated = `a default secret key has been generated. If you want to override it, you can set the HTTP_SECRET environment variable.`
)

var errServerReset = errors.New("server_reset")

type (
	// Configuration options needed by the server to handle request correctly.
	Options interface {
		docker.Options
		raw.Options
		git.Options
		archive.Options

		IsVerbose() bool
		ConnectionString() string
		Secret() []byte
		UseSSL() bool
		ListenAddress() string
		DefaultEmail() string
		DefaultPassword() string
		IsUsingGeneratedSecret() bool
		DataDir() string
		LogsDir() string
		ConfigPath() string
		CurrentVersion() string
		DeploymentDirTemplate() domain.DeploymentDirTemplate
	}

	server struct {
		// Services

		options     Options
		router      *gin.Engine
		logger      log.Logger
		db          sqlite.Database
		docker      docker.Backend
		usersReader authdomain.UsersReader
		worker      async.Worker

		// Commands & Queries
		// FIXME: maybe there is a way to introduce a mediator or something like that in the future

		authGateway            authquery.Gateway
		deploymentGateway      deplquery.Gateway
		login                  func(context.Context, authcmd.LoginCommand) (string, error)
		createFirstAccount     func(context.Context, authcmd.CreateFirstAccountCommand) error
		updateUser             func(context.Context, authcmd.UpdateUserCommand) error
		createApp              func(context.Context, deplcmd.CreateAppCommand) (string, error)
		updateApp              func(context.Context, deplcmd.UpdateAppCommand) error
		requestAppCleanup      func(context.Context, deplcmd.RequestAppCleanupCommand) error
		queueDeployment        func(context.Context, deplcmd.QueueDeploymentCommand) (int, error)
		redeploy               func(context.Context, deplcmd.RedeployCommand) (int, error)
		promote                func(context.Context, deplcmd.PromoteCommand) (int, error)
		failRunningDeployments func(context.Context, error) error
		queueJob               func(context.Context, workercmd.QueueCommand) error
	}
)

func newHttpServer(options Options) (*server, error) {
	s := &server{options: options}

	if err := s.configureServices(); err != nil {
		return nil, err
	}

	s.configureRouter()

	return s, nil
}

func (s *server) Cleanup() {
	s.logger.Debug("cleaning server services")

	s.worker.Stop()

	if err := s.db.Close(); err != nil {
		panic(err)
	}
}

func (s *server) Listen() (finalErr error) {
	if err := s.startCheckup(); err != nil {
		return err
	}

	srv := &http.Server{
		Addr:    s.options.ListenAddress(),
		Handler: s.router,
	}

	s.logger.Infow("launching web server",
		"address", srv.Addr,
	)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			finalErr = err
		}
	}()

	// Let's handle the graceful shutdown of the http server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	s.logger.Info("shutting down the web server, please wait")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		finalErr = err
	}

	return finalErr
}

func (s *server) Logger() log.Logger { return s.logger }
func (s *server) UseSSL() bool       { return s.options.UseSSL() }

// Configure every services needed by this service. Think of it as the composition root.
func (s *server) configureServices() error {
	// Prepare the logger
	s.logger = log.NewLogger(s.options.IsVerbose())

	s.logger.Infow("configuration loaded", "path", s.options.ConfigPath())

	// Prepare the database
	dispatcher := event.NewInProcessDispatcher(s.domainEventHandler)
	db, err := sqlite.Open(s.options.ConnectionString(), s.logger, dispatcher)

	if err != nil {
		return err
	}

	s.db = db

	// Prepare needed services

	usersStore := authsqlite.NewUsersStore(s.db)
	appsStore := deplsqlite.NewAppsStore(s.db)
	deploymentsStore := deplsqlite.NewDeploymentsStore(s.db)
	jobsStore := workersqlite.NewJobsStore(s.db)

	triggerFacade := trigger.NewFacade(
		raw.New(s.options),
		archive.New(s.options),
		git.New(s.options, appsStore),
	)

	s.docker = docker.New(s.options, s.logger)
	handler := workerinfra.NewHandlerFacade(s.logger,
		jobs.DeploymentHandler(s.logger, deplcmd.Deploy(deploymentsStore, deploymentsStore, triggerFacade, s.docker)),
		jobs.CleanupAppHandler(s.logger, deplcmd.CleanupApp(deploymentsStore, appsStore, appsStore, s.docker)),
	)
	s.worker = async.NewIntervalWorker(s.logger, 5*time.Second, workercmd.ProcessNext(jobsStore, jobsStore, handler))
	s.usersReader = usersStore

	// Queries

	s.authGateway = authsqlite.NewGateway(s.db)
	s.deploymentGateway = deplsqlite.NewGateway(s.db)

	// And finally commands!

	passwordHasher := authinfra.NewBCryptHasher()

	s.login = authcmd.Login(usersStore, passwordHasher)
	s.createFirstAccount = authcmd.CreateFirstAccount(usersStore, usersStore, passwordHasher, authinfra.NewKeyGenerator())
	s.updateUser = authcmd.UpdateUser(usersStore, usersStore, passwordHasher)

	s.createApp = deplcmd.CreateApp(appsStore, appsStore)
	s.updateApp = deplcmd.UpdateApp(appsStore, appsStore)
	s.requestAppCleanup = deplcmd.RequestAppCleanup(appsStore, appsStore)
	s.queueDeployment = deplcmd.QueueDeployment(appsStore, deploymentsStore, deploymentsStore, triggerFacade, s.options.DeploymentDirTemplate())
	s.redeploy = deplcmd.Redeploy(appsStore, deploymentsStore, deploymentsStore, s.options.DeploymentDirTemplate())
	s.promote = deplcmd.Promote(appsStore, deploymentsStore, deploymentsStore, s.options.DeploymentDirTemplate())
	s.failRunningDeployments = deplcmd.FailRunningDeployments(deploymentsStore, deploymentsStore)

	s.queueJob = workercmd.Queue(jobsStore)

	return nil
}

// Do some startup checks to ensure everything is ready before launching the web server.
func (s *server) startCheckup() error {
	if s.options.IsUsingGeneratedSecret() {
		s.logger.Info(noticeSecretKeyGenerated)
	}

	// Migrate the database first
	if err := s.db.Migrate(sqlite.MigrationsDir{
		"auth":       authsqlite.Migrations,
		"deployment": deplsqlite.Migrations,
		"worker":     workersqlite.Migrations,
	}); err != nil {
		return err
	}

	// Checks for unsupported configuration modifications
	if err := s.checkNonSupportedConfigChanges(); err != nil {
		return err
	}

	ctx := context.Background()

	// Fail stale deployments
	if err := s.failRunningDeployments(ctx, errServerReset); err != nil {
		return err
	}

	// Create an admin account if no one exists yet
	if err := s.createFirstAccount(ctx, authcmd.CreateFirstAccountCommand{
		Email:    s.options.DefaultEmail(),
		Password: s.options.DefaultPassword(),
	}); err != nil {
		return err
	}

	if err := s.docker.Setup(); err != nil {
		return err
	}

	go s.worker.Start()

	return nil
}

func (s *server) configureRouter() {
	if !s.options.IsVerbose() {
		gin.SetMode(gin.ReleaseMode)
	}

	s.router = gin.Default()
	s.router.SetTrustedProxies(nil)

	// Configure the session store
	store := cookie.NewStore(s.options.Secret())
	store.Options(sessions.Options{
		Secure:   s.options.UseSSL(),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	s.router.Use(sessions.Sessions(sessionName, store), s.transactional)

	// Let's register every routes now!
	v1 := s.router.Group("/api/v1")

	// Public routes
	v1.POST("/sessions", s.createSessionHandler())
	v1.GET("/healthcheck", s.healthcheckHandler)

	// Authenticated routes
	v1secured := v1.Group("", s.authenticate(false))
	v1secured.DELETE("/session", s.deleteSessionHandler())
	v1secured.GET("/users/:id", s.getUserByIDHandler())
	v1secured.GET("/users", s.listUsersHandler())

	v1securedAllowApi := v1.Group("", s.authenticate(true))
	v1securedAllowApi.GET("/profile", s.getProfileHandler())
	v1securedAllowApi.PATCH("/profile", s.updateProfileHandler())
	v1securedAllowApi.GET("/apps", s.listAppsHandler())
	v1securedAllowApi.POST("/apps", s.createAppHandler())
	v1securedAllowApi.PATCH("/apps/:id", s.updateAppHandler())
	v1securedAllowApi.DELETE("/apps/:id", s.requestAppCleanupHandler())
	v1securedAllowApi.GET("/apps/:id", s.getAppByIDHandler())
	v1securedAllowApi.POST("/apps/:id/deployments", s.queueDeploymentHandler())
	v1securedAllowApi.GET("/apps/:id/deployments", s.listDeploymentsByAppHandler())
	v1securedAllowApi.GET("/apps/:id/deployments/:number", s.getDeploymentByIDHandler())
	v1securedAllowApi.POST("/apps/:id/deployments/:number/redeploy", s.redeployHandler())
	v1securedAllowApi.POST("/apps/:id/deployments/:number/promote", s.promoteHandler())
	v1securedAllowApi.GET("/apps/:id/deployments/:number/logs", s.getDeploymentLogsHandler())

	if s.options.IsVerbose() {
		pprof.Register(s.router)
	}

	s.useSPA()
}

// Main handler which listen for specific domain events raised by core domains.
// This handler is hardcoded to prevent having to define a Name on each event to determine
// which handlers to call (something like worker.Handler for example).
//
// I find it easier for now.
func (s *server) domainEventHandler(ctx context.Context, e event.Event) error {
	switch evt := e.(type) {
	case domain.DeploymentCreated:
		s.queueJob(ctx, jobs.Deployment(evt.ID))
	case domain.AppCleanupRequested:
		s.queueJob(ctx, jobs.CleanupApp(evt.ID))
	}

	return nil
}

func (s *server) useSPA() {
	// Retrieve the root build directory
	frontendRootDir, _ := fs.Sub(front, embeddedRootDir)
	// Wrap it in an HTTP filesystem
	frontendFS := http.FS(frontendRootDir)

	// And serve static files
	s.router.Use(func(ctx *gin.Context) {
		filepath := ctx.Request.URL.Path

		// If it has a trailing slash, it should be a pretty url so append "index.html"
		if strings.HasSuffix(filepath, "/") {
			filepath = path.Join(filepath, "index.html")
		}

		// Check if the file exists
		file, err := frontendFS.Open(filepath)

		// If it could not be found, fallback to the fallback.html file and let the SPA routes the request
		if os.IsNotExist(err) {
			ctx.FileFromFS("/fallback.html", frontendFS)
			return
		}

		if err == nil {
			// File was found, if it was a fingerprinted asset, add a cache control header.
			extension := path.Ext(filepath)

			switch extension {
			case ".css", ".js":
				ctx.Header("Cache-Control", "max-age=31536000, immutable")
			}

			file.Close()
		}

		ctx.FileFromFS(ctx.Request.URL.Path, frontendFS)
	})
}

func (s *server) checkNonSupportedConfigChanges() error {
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
