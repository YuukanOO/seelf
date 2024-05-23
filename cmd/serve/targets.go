package serve

import (
	"github.com/YuukanOO/seelf/internal/deployment/app/create_target"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_target"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_targets"
	"github.com/YuukanOO/seelf/internal/deployment/app/reconfigure_target"
	"github.com/YuukanOO/seelf/internal/deployment/app/request_target_cleanup"
	"github.com/YuukanOO/seelf/internal/deployment/app/update_target"
	"github.com/YuukanOO/seelf/internal/deployment/infra/provider/docker"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/http"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/gin-gonic/gin"
)

type createTargetBody struct {
	create_target.Command

	Docker monad.Maybe[docker.Body] `json:"docker"`
}

func (s *server) createTargetHandler() gin.HandlerFunc {
	return http.Bind(s, func(c *gin.Context, body createTargetBody) error {
		ctx := c.Request.Context()

		if dockerBody, isSet := body.Docker.TryGet(); isSet {
			body.Provider = dockerBody
		}

		id, err := bus.Send(s.bus, ctx, body.Command)

		if err != nil {
			return err
		}

		target, err := bus.Send(s.bus, ctx, get_target.Query{
			ID: id,
		})

		if err != nil {
			return err
		}

		return http.Created(s, c, target, "/api/v1/targets/%s", id)
	})
}

type updateTargetBody struct {
	update_target.Command

	Docker monad.Maybe[docker.Body] `json:"docker"`
}

func (s *server) updateTargetHandler() gin.HandlerFunc {
	return http.Bind(s, func(c *gin.Context, body updateTargetBody) error {
		ctx := c.Request.Context()

		body.ID = c.Param("id")

		if dockerBody, isSet := body.Docker.TryGet(); isSet {
			body.Provider = dockerBody
		}

		id, err := bus.Send(s.bus, ctx, body.Command)

		if err != nil {
			return err
		}

		data, err := bus.Send(s.bus, ctx, get_target.Query{
			ID: id,
		})

		if err != nil {
			return err
		}

		return http.Ok(c, data)
	})
}

func (s *server) reconfigureTargetHandler() gin.HandlerFunc {
	return http.Send(s, func(c *gin.Context) error {
		_, err := bus.Send(s.bus, c.Request.Context(), reconfigure_target.Command{
			ID: c.Param("id"),
		})

		if err != nil {
			return err
		}

		return http.NoContent(c)
	})
}

func (s *server) deleteTargetHandler() gin.HandlerFunc {
	return http.Send(s, func(c *gin.Context) error {
		_, err := bus.Send(s.bus, c.Request.Context(), request_target_cleanup.Command{
			ID: c.Param("id"),
		})

		if err != nil {
			return err
		}

		return http.NoContent(c)
	})
}

func (s *server) listTargetsHandler() gin.HandlerFunc {
	return http.Bind(s, func(c *gin.Context, query get_targets.Query) error {
		targets, err := bus.Send(s.bus, c.Request.Context(), query)

		if err != nil {
			return err
		}

		return http.Ok(c, targets)
	})
}

func (s *server) getTargetByIDHandler() gin.HandlerFunc {
	return http.Send(s, func(c *gin.Context) error {
		target, err := bus.Send(s.bus, c.Request.Context(), get_target.Query{
			ID: c.Param("id"),
		})

		if err != nil {
			return err
		}

		return http.Ok(c, target)
	})
}
