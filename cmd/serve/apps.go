package serve

import (
	"github.com/YuukanOO/seelf/internal/deployment/app/command"
	"github.com/YuukanOO/seelf/pkg/http"
	"github.com/gin-gonic/gin"
)

func (s *server) createAppHandler() gin.HandlerFunc {
	return http.Bind(s, func(ctx *gin.Context, cmd command.CreateAppCommand) error {
		context := ctx.Request.Context()
		appid, err := s.createApp(context, cmd)

		if err != nil {
			return err
		}

		data, err := s.deploymentGateway.GetAppByID(context, appid)

		if err != nil {
			return err
		}

		return http.Created(s, ctx, data, "/api/v1/apps/%s", appid)
	})
}

func (s *server) updateAppHandler() gin.HandlerFunc {
	return http.Bind(s, func(ctx *gin.Context, cmd command.UpdateAppCommand) error {
		cmd.ID = ctx.Param("id")
		context := ctx.Request.Context()

		if err := s.updateApp(context, cmd); err != nil {
			return err
		}

		data, err := s.deploymentGateway.GetAppByID(context, cmd.ID)

		if err != nil {
			return err
		}

		return http.Ok(ctx, data)
	})
}

func (s *server) listAppsHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		apps, err := s.deploymentGateway.GetAllApps(ctx.Request.Context())

		if err != nil {
			return err
		}

		return http.Ok(ctx, apps)
	})
}

func (s *server) getAppByIDHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		app, err := s.deploymentGateway.GetAppByID(ctx.Request.Context(), ctx.Param("id"))

		if err != nil {
			return err
		}

		return http.Ok(ctx, app)
	})
}

func (s *server) requestAppCleanupHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		cmd := command.RequestAppCleanupCommand{
			ID: ctx.Param("id"),
		}

		if err := s.requestAppCleanup(ctx.Request.Context(), cmd); err != nil {
			return err
		}

		return http.NoContent(ctx)
	})
}
