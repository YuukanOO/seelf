package serve

import (
	"github.com/YuukanOO/seelf/internal/auth/app/command"
	"github.com/YuukanOO/seelf/pkg/http"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func (s *server) createSessionHandler() gin.HandlerFunc {
	return http.Bind(s, func(ctx *gin.Context, cmd command.LoginCommand) error {
		sess := sessions.Default(ctx)
		context := ctx.Request.Context()
		uid, err := s.login(context, cmd)

		if err != nil {
			return err
		}

		// Everything went good, let's set the user cookie
		sess.Set(userSessionKey, uid)

		if err = sess.Save(); err != nil {
			return err
		}

		user, err := s.authGateway.GetProfile(context, uid)

		if err != nil {
			return err
		}

		return http.Created(s, ctx, user, "/api/v1/session")
	})
}

func (s *server) deleteSessionHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		sess := sessions.Default(ctx)

		sess.Clear()

		if err := sess.Save(); err != nil {
			return err
		}

		return http.NoContent(ctx)
	})
}
