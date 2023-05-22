package serve

import (
	"github.com/YuukanOO/seelf/internal/auth/app/command"
	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/http"
	"github.com/gin-gonic/gin"
)

func (s *server) updateProfileHandler() gin.HandlerFunc {
	return http.Bind(s, func(ctx *gin.Context, cmd command.UpdateUserCommand) error {
		c := ctx.Request.Context()
		cmd.ID = string(domain.CurrentUser(c).MustGet())

		if err := s.updateUser(c, cmd); err != nil {
			return err
		}

		user, err := s.authGateway.GetProfile(c, cmd.ID)

		if err != nil {
			return err
		}

		return http.Ok(ctx, user)
	})
}

func (s *server) listUsersHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		users, err := s.authGateway.GetAllUsers(ctx.Request.Context())

		if err != nil {
			return err
		}

		return http.Ok(ctx, users)
	})
}

func (s *server) getProfileHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		c := ctx.Request.Context()
		currentUserID := domain.CurrentUser(c).MustGet()
		user, err := s.authGateway.GetProfile(c, string(currentUserID))

		if err != nil {
			return err
		}

		return http.Ok(ctx, user)
	})
}

func (s *server) getUserByIDHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		user, err := s.authGateway.GetUserByID(ctx.Request.Context(), ctx.Param("id"))

		if err != nil {
			return err
		}

		return http.Ok(ctx, user)
	})
}
