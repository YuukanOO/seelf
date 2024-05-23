package serve

import (
	"github.com/YuukanOO/seelf/internal/deployment/app/create_registry"
	"github.com/YuukanOO/seelf/internal/deployment/app/delete_registry"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_registries"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_registry"
	"github.com/YuukanOO/seelf/internal/deployment/app/update_registry"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/http"
	"github.com/gin-gonic/gin"
)

func (s *server) createRegistryHandler() gin.HandlerFunc {
	return http.Bind(s, func(c *gin.Context, cmd create_registry.Command) error {
		ctx := c.Request.Context()

		id, err := bus.Send(s.bus, ctx, cmd)

		if err != nil {
			return err
		}

		data, err := bus.Send(s.bus, ctx, get_registry.Query{
			ID: id,
		})

		if err != nil {
			return err
		}

		return http.Created(s, c, data, "/api/v1/registries/%s", id)
	})
}

func (s *server) updateRegistryHandler() gin.HandlerFunc {
	return http.Bind(s, func(c *gin.Context, cmd update_registry.Command) error {
		cmd.ID = c.Param("id")
		ctx := c.Request.Context()

		id, err := bus.Send(s.bus, ctx, cmd)

		if err != nil {
			return err
		}

		data, err := bus.Send(s.bus, ctx, get_registry.Query{
			ID: id,
		})

		if err != nil {
			return err
		}

		return http.Ok(c, data)
	})
}

func (s *server) deleteRegistryHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		if _, err := bus.Send(s.bus, ctx.Request.Context(), delete_registry.Command{
			ID: ctx.Param("id"),
		}); err != nil {
			return err
		}

		return http.NoContent(ctx)
	})
}

func (s *server) listRegistriesHandler() gin.HandlerFunc {
	return http.Send(s, func(c *gin.Context) error {
		data, err := bus.Send(s.bus, c.Request.Context(), get_registries.Query{})

		if err != nil {
			return err
		}

		return http.Ok(c, data)
	})
}

func (s *server) getRegistryByIDHandler() gin.HandlerFunc {
	return http.Send(s, func(c *gin.Context) error {
		data, err := bus.Send(s.bus, c.Request.Context(), get_registry.Query{
			ID: c.Param("id"),
		})

		if err != nil {
			return err
		}

		return http.Ok(c, data)
	})
}
