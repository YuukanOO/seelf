package serve

import (
	"github.com/YuukanOO/seelf/internal/auth/app/get_profile"
	"github.com/YuukanOO/seelf/internal/auth/app/login"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/http"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func (s *server) createSessionHandler() gin.HandlerFunc {
	return http.Bind(s, func(ctx *gin.Context, cmd login.Command) error {
		sess := sessions.Default(ctx)
		context := ctx.Request.Context()
		uid, err := bus.Send(s.bus, context, cmd)

		if err != nil {
			return err
		}

		// Everything went good, let's set the user cookie
		sess.Set(userSessionKey, uid)

		if err = sess.Save(); err != nil {
			return err
		}

		user, err := bus.Send(s.bus, context, get_profile.Query{
			ID: uid,
		})

		if err != nil {
			return err
		}

		return http.Created(s, ctx, user, "/api/v1/profile")
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
