package serve

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/YuukanOO/seelf/cmd/startup"
	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

//go:embed all:front/build/*
var front embed.FS

const (
	embeddedRootDir = "front/build"
	sessionName     = "seelf"
)

type (
	// Configuration options needed by the server to handle request correctly.
	ServerOptions interface {
		Secret() []byte
		IsSecure() bool
		ListenAddress() string
	}

	server struct {
		options            ServerOptions
		router             *gin.Engine
		bus                bus.Dispatcher
		logger             log.Logger
		usersReader        domain.UsersReader
		scheduledJobsStore bus.ScheduledJobsStore
	}
)

func newHttpServer(options ServerOptions, root startup.ServerRoot) *server {
	gin.SetMode(gin.ReleaseMode)

	s := &server{
		options:            options,
		router:             gin.New(),
		usersReader:        root.UsersReader(),
		scheduledJobsStore: root.ScheduledJobsStore(),
		bus:                root.Bus(),
		logger:             root.Logger(),
	}

	s.router.SetTrustedProxies(nil)

	// Configure the session store
	store := cookie.NewStore(s.options.Secret())
	store.Options(sessions.Options{
		Secure:   s.options.IsSecure(),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	s.router.Use(s.requestLogger, s.recoverer, sessions.Sessions(sessionName, store))

	// Let's register every routes now!
	v1 := s.router.Group("/api/v1")

	// Public routes
	v1.POST("/sessions", s.createSessionHandler())
	v1.GET("/healthcheck", s.healthcheckHandler)

	// Authenticated routes
	v1secured := v1.Group("", s.authenticate(false))
	v1secured.DELETE("/session", s.deleteSessionHandler())
	v1secured.GET("/jobs", s.listJobsHandler())
	v1secured.DELETE("/jobs/:id", s.deleteJobsHandler())
	v1secured.GET("/profile", s.getProfileHandler())
	v1secured.PATCH("/profile", s.updateProfileHandler())
	v1secured.POST("/targets", s.createTargetHandler())
	v1secured.PATCH("/targets/:id", s.updateTargetHandler())
	v1secured.POST("/targets/:id/reconfigure", s.reconfigureTargetHandler())
	v1secured.GET("/targets", s.listTargetsHandler())
	v1secured.GET("/targets/:id", s.getTargetByIDHandler())
	v1secured.DELETE("/targets/:id", s.deleteTargetHandler())
	v1secured.GET("/apps", s.listAppsHandler())
	v1secured.POST("/apps", s.createAppHandler())
	v1secured.PATCH("/apps/:id", s.updateAppHandler())
	v1secured.DELETE("/apps/:id", s.requestAppCleanupHandler())

	// Allow API Key authentication for those routes
	// FIXME: in the future, maybe all the API should be accessible, but not before https://github.com/YuukanOO/seelf/issues/45
	v1securedAllowApi := v1.Group("", s.authenticate(true))
	v1securedAllowApi.GET("/apps/:id", s.getAppByIDHandler())
	v1securedAllowApi.POST("/apps/:id/deployments", s.queueDeploymentHandler())
	v1securedAllowApi.GET("/apps/:id/deployments", s.listDeploymentsByAppHandler())
	v1securedAllowApi.GET("/apps/:id/deployments/:number", s.getDeploymentByIDHandler())
	v1securedAllowApi.POST("/apps/:id/deployments/:number/redeploy", s.redeployHandler())
	v1securedAllowApi.POST("/apps/:id/deployments/:number/promote", s.promoteHandler())
	v1securedAllowApi.GET("/apps/:id/deployments/:number/logs", s.getDeploymentLogsHandler())

	s.useSPA()

	return s
}

func (s *server) Listen() (finalErr error) {
	srv := &http.Server{
		Addr:    s.options.ListenAddress(),
		Handler: s.router,
	}

	s.logger.Infow("launching web server",
		"address", srv.Addr,
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			finalErr = err
			quit <- syscall.SIGTERM
		}
	}()

	// Let's handle the graceful shutdown of the http server
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
func (s *server) IsSecure() bool     { return s.options.IsSecure() }

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
