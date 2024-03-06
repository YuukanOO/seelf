package serve

import (
	"github.com/YuukanOO/seelf/internal/deployment/app/create_target"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_target"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_targets"
	"github.com/YuukanOO/seelf/internal/deployment/app/request_target_delete"
	"github.com/YuukanOO/seelf/internal/deployment/infra/provider/docker"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/http"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/gin-gonic/gin"
)

type createTargetBody struct {
	Name   string                   `json:"name"`
	Domain string                   `json:"domain"`
	Docker monad.Maybe[docker.Body] `json:"docker"`
}

func (s *server) createTargetHandler() gin.HandlerFunc {
	return http.Bind(s, func(c *gin.Context, body createTargetBody) error {
		var (
			payload any
			ctx     = c.Request.Context()
		)

		if dockerBody, isSet := body.Docker.TryGet(); isSet {
			payload = dockerBody
		}

		id, err := bus.Send(s.bus, ctx, create_target.Command{
			Name:     body.Name,
			Domain:   body.Domain,
			Provider: payload,
		})

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

func (s *server) deleteTargetHandler() gin.HandlerFunc {
	return http.Send(s, func(c *gin.Context) error {
		_, err := bus.Send(s.bus, c.Request.Context(), request_target_delete.Command{
			ID: c.Param("id"),
		})

		if err != nil {
			return err
		}

		return http.NoContent(c)
	})
}

func (s *server) listTargetsHandler() gin.HandlerFunc {
	return http.Send(s, func(c *gin.Context) error {
		targets, err := bus.Send(s.bus, c.Request.Context(), get_targets.Query{})

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
