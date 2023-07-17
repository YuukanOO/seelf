package serve

import (
	"mime/multipart"
	"path/filepath"
	"strconv"

	"github.com/YuukanOO/seelf/internal/deployment/app/command"
	"github.com/YuukanOO/seelf/internal/deployment/app/query"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/git"
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
	Git         monad.Maybe[git.Payload] `json:"git"`
}

func (s *server) queueDeploymentHandler() gin.HandlerFunc {
	return http.Bind(s, func(ctx *gin.Context, body queueDeploymentBody) error {
		var (
			payload any
			context = ctx.Request.Context()
			appid   = ctx.Param("id")
		)

		// Resolve the payload data.
		if body.Git.HasValue() {
			payload = body.Git.MustGet()
		} else if body.Raw.HasValue() {
			payload = body.Raw.MustGet()
		} else if body.Archive != nil {
			payload = body.Archive
		}

		number, err := s.queueDeployment(context, command.QueueDeploymentCommand{
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
			context   = ctx.Request.Context()
			appid     = ctx.Param("id")
			source, _ = strconv.Atoi(ctx.Param("number"))
		)

		number, err := s.redeploy(context, command.RedeployCommand{
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
			context   = ctx.Request.Context()
			appid     = ctx.Param("id")
			source, _ = strconv.Atoi(ctx.Param("number"))
		)

		number, err := s.promote(context, command.PromoteCommand{
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

		deployment, err := s.deploymentGateway.GetDeploymentByID(ctx.Request.Context(), ctx.Param("id"), number)

		if err != nil {
			return err
		}

		return http.Ok(ctx, deployment)
	})
}

func (s *server) getDeploymentLogsHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		number, _ := strconv.Atoi(ctx.Param("number"))
		logfile, err := s.deploymentGateway.GetDeploymentLogfileByID(ctx.Request.Context(), ctx.Param("id"), number)

		if err != nil {
			return err
		}

		return http.File(ctx, filepath.Join(s.options.LogsDir(), logfile))
	})
}

// FIXME: till gin support custom types in query binding...
type getDeploymentsFilters struct {
	Page        int    `form:"page"`
	Environment string `form:"environment"`
}

func (s *server) listDeploymentsByAppHandler() gin.HandlerFunc {
	return http.Bind(s, func(ctx *gin.Context, request getDeploymentsFilters) error {
		var filters query.GetDeploymentsFilters

		if request.Environment != "" {
			filters.Environment = filters.Environment.WithValue(request.Environment)
		}

		if request.Page != 0 {
			filters.Page = filters.Page.WithValue(request.Page)
		}

		deployments, err := s.deploymentGateway.GetAllDeploymentsByApp(ctx.Request.Context(), ctx.Param("id"), filters)

		if err != nil {
			return err
		}

		return http.Ok(ctx, deployments)
	})
}

func (s *server) sendDeploymentCreatedResponse(ctx *gin.Context, appid string, number int) error {
	deployment, err := s.deploymentGateway.GetDeploymentByID(ctx.Request.Context(), appid, number)

	if err != nil {
		return err
	}

	return http.Created(s, ctx, deployment, "/api/v1/apps/%s/deployments/%d", appid, number)
}
