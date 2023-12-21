package serve

import (
	"mime/multipart"
	"strconv"

	"github.com/YuukanOO/seelf/internal/deployment/app/get_app_deployments"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_deployment"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_deployment_log"
	"github.com/YuukanOO/seelf/internal/deployment/app/promote"
	"github.com/YuukanOO/seelf/internal/deployment/app/queue_deployment"
	"github.com/YuukanOO/seelf/internal/deployment/app/redeploy"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/git"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/http"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/gin-gonic/gin"
)

// Specific body for the queue deployment endpoint. Will resolve to the appropriate payload
// based on the provided fields.
type queueDeploymentBody struct {
	Environment string                   `json:"environment" form:"environment"`
	Raw         monad.Maybe[string]      `json:"raw"`
	Archive     *multipart.FileHeader    `form:"archive"`
	Git         monad.Maybe[git.Request] `json:"git"`
}

func (s *server) queueDeploymentHandler() gin.HandlerFunc {
	return http.Bind(s, func(ctx *gin.Context, body queueDeploymentBody) error {
		var (
			payload any
			context = ctx.Request.Context()
			appid   = ctx.Param("id")
		)

		// Resolve the payload data.
		if gitBody, isSet := body.Git.TryGet(); isSet {
			payload = gitBody
		} else if rawBody, isSet := body.Raw.TryGet(); isSet {
			payload = rawBody
		} else if body.Archive != nil {
			payload = body.Archive
		}

		number, err := bus.Send(s.bus, context, queue_deployment.Command{
			AppID:       appid,
			Environment: body.Environment,
			Payload:     payload,
		})

		if err != nil {
			return err
		}

		return s.sendDeploymentCreatedResponse(ctx, appid, number)
	})
}

func (s *server) redeployHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		var (
			appid     = ctx.Param("id")
			source, _ = strconv.Atoi(ctx.Param("number"))
		)

		number, err := bus.Send(s.bus, ctx.Request.Context(), redeploy.Command{
			AppID:            appid,
			DeploymentNumber: source,
		})

		if err != nil {
			return err
		}

		return s.sendDeploymentCreatedResponse(ctx, appid, number)
	})
}

func (s *server) promoteHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		var (
			appid     = ctx.Param("id")
			source, _ = strconv.Atoi(ctx.Param("number"))
		)

		number, err := bus.Send(s.bus, ctx.Request.Context(), promote.Command{
			AppID:            appid,
			DeploymentNumber: source,
		})

		if err != nil {
			return err
		}

		return s.sendDeploymentCreatedResponse(ctx, appid, number)
	})
}

func (s *server) getDeploymentByIDHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		number, _ := strconv.Atoi(ctx.Param("number"))

		deployment, err := bus.Send(s.bus, ctx.Request.Context(), get_deployment.Query{
			AppID:            ctx.Param("id"),
			DeploymentNumber: number,
		})

		if err != nil {
			return err
		}

		return http.Ok(ctx, deployment)
	})
}

func (s *server) getDeploymentLogsHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		appid := ctx.Param("id")
		number, _ := strconv.Atoi(ctx.Param("number"))

		logpath, err := bus.Send(s.bus, ctx.Request.Context(), get_deployment_log.Query{
			AppID:            appid,
			DeploymentNumber: number,
		})

		if err != nil {
			return err
		}

		return http.File(ctx, logpath)
	})
}

// FIXME: till gin support custom types in query binding...
type getDeploymentsFilters struct {
	Page        int    `form:"page"`
	Environment string `form:"environment"`
}

func (s *server) listDeploymentsByAppHandler() gin.HandlerFunc {
	return http.Bind(s, func(ctx *gin.Context, request getDeploymentsFilters) error {
		query := get_app_deployments.Query{
			AppID: ctx.Param("id"),
		}

		if request.Environment != "" {
			query.Environment = query.Environment.WithValue(request.Environment)
		}

		if request.Page != 0 {
			query.Page = query.Page.WithValue(request.Page)
		}

		deployments, err := bus.Send(s.bus, ctx.Request.Context(), query)

		if err != nil {
			return err
		}

		return http.Ok(ctx, deployments)
	})
}

func (s *server) sendDeploymentCreatedResponse(ctx *gin.Context, appid string, number int) error {
	deployment, err := bus.Send(s.bus, ctx.Request.Context(), get_deployment.Query{
		AppID:            appid,
		DeploymentNumber: number,
	})

	if err != nil {
		return err
	}

	return http.Created(s, ctx, deployment, "/api/v1/apps/%s/deployments/%d", appid, number)
}
