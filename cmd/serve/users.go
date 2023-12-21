package serve

import (
	"github.com/YuukanOO/seelf/internal/auth/app/get_profile"
	"github.com/YuukanOO/seelf/internal/auth/app/update_user"
	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/http"
	"github.com/gin-gonic/gin"
)

func (s *server) updateProfileHandler() gin.HandlerFunc {
	return http.Bind(s, func(ctx *gin.Context, cmd update_user.Command) error {
		c := ctx.Request.Context()
		cmd.ID = string(domain.CurrentUser(c).MustGet())

		if _, err := bus.Send(s.bus, c, cmd); err != nil {
			return err
		}

		user, err := bus.Send(s.bus, c, get_profile.Query{
			ID: cmd.ID,
		})

		if err != nil {
			return err
		}

		return http.Ok(ctx, user)
	})
}

func (s *server) getProfileHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		c := ctx.Request.Context()
		currentUserID := domain.CurrentUser(c).MustGet()
		user, err := bus.Send(s.bus, c, get_profile.Query{
			ID: string(currentUserID),
		})

		if err != nil {
			return err
		}

		return http.Ok(ctx, user)
	})
}
