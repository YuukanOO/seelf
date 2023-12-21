package serve

import (
	"github.com/YuukanOO/seelf/internal/deployment/app/create_app"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_app_detail"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_apps"
	"github.com/YuukanOO/seelf/internal/deployment/app/request_app_cleanup"
	"github.com/YuukanOO/seelf/internal/deployment/app/update_app"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/http"
	"github.com/gin-gonic/gin"
)

func (s *server) createAppHandler() gin.HandlerFunc {
	return http.Bind(s, func(ctx *gin.Context, cmd create_app.Command) error {
		context := ctx.Request.Context()
		appid, err := bus.Send(s.bus, context, cmd)

		if err != nil {
			return err
		}

		data, err := bus.Send(s.bus, context, get_app_detail.Query{
			ID: appid,
		})

		if err != nil {
			return err
		}

		return http.Created(s, ctx, data, "/api/v1/apps/%s", appid)
	})
}

func (s *server) updateAppHandler() gin.HandlerFunc {
	return http.Bind(s, func(ctx *gin.Context, cmd update_app.Command) error {
		cmd.ID = ctx.Param("id")
		context := ctx.Request.Context()

		if _, err := bus.Send(s.bus, context, cmd); err != nil {
			return err
		}

		data, err := bus.Send(s.bus, context, get_app_detail.Query{
			ID: cmd.ID,
		})

		if err != nil {
			return err
		}

		return http.Ok(ctx, data)
	})
}

func (s *server) listAppsHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		apps, err := bus.Send(s.bus, ctx.Request.Context(), get_apps.Query{})

		if err != nil {
			return err
		}

		return http.Ok(ctx, apps)
	})
}

func (s *server) getAppByIDHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		app, err := bus.Send(s.bus, ctx.Request.Context(), get_app_detail.Query{
			ID: ctx.Param("id"),
		})

		if err != nil {
			return err
		}

		return http.Ok(ctx, app)
	})
}

func (s *server) requestAppCleanupHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		cmd := request_app_cleanup.Command{
			ID: ctx.Param("id"),
		}

		if _, err := bus.Send(s.bus, ctx.Request.Context(), cmd); err != nil {
			return err
		}

		return http.NoContent(ctx)
	})
}
