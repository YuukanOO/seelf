package serve

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/YuukanOO/seelf/internal/auth/app/api_login"
	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	httputils "github.com/YuukanOO/seelf/pkg/http"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	userSessionKey      = "seelf-session"
	apiAuthHeader       = "Authorization"
	apiAuthPrefix       = "Bearer "
	apiAuthPrefixLength = len(apiAuthPrefix)
)

var errUnauthorized = errors.New("unauthorized")

func (s *server) authenticate(withApiAccess bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// First, try to find a user id in the encrypted session cookie
		userSession := sessions.Default(ctx)
		uid, ok := userSession.Get(userSessionKey).(string)
		failed := !ok || uid == ""

		// If it failed and api access is not allowed, return early
		if failed && !withApiAccess {
			_ = ctx.AbortWithError(http.StatusUnauthorized, errUnauthorized)
			return
		}

		// Else, if we do not fail, the user id has been found, go on
		if !failed {
			ctx.Request = ctx.Request.WithContext(domain.WithUserID(ctx.Request.Context(), domain.UserID(uid)))
			ctx.Next()
			return
		}

		// If we are here, look in the request header to check if an api key is present and check if it corresponds to an existing user
		authHeader := ctx.GetHeader(apiAuthHeader)

		if !strings.HasPrefix(authHeader, apiAuthPrefix) {
			_ = ctx.AbortWithError(http.StatusUnauthorized, errUnauthorized)
			return
		}

		id, err := bus.Send(s.bus, ctx.Request.Context(), api_login.Query{
			Key: authHeader[apiAuthPrefixLength:],
		})

		if err != nil {
			_ = ctx.AbortWithError(http.StatusUnauthorized, errUnauthorized)
			return
		}

		// Attach the user id to the context passed down in every use cases.
		ctx.Request = ctx.Request.WithContext(domain.WithUserID(ctx.Request.Context(), domain.UserID(id)))

		ctx.Next()
	}
}

func (s *server) requestLogger(ctx *gin.Context) {
	defer func(start time.Time, c *gin.Context) {
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		if raw != "" {
			path = path + "?" + raw
		}

		s.logger.Debugw(path,
			"status", c.Writer.Status(),
			"method", c.Request.Method,
			"elapsed", time.Since(start))
	}(time.Now(), ctx)

	ctx.Next()
}

func (s *server) recoverer(ctx *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			asErr, ok := err.(error)

			if !ok {
				asErr = fmt.Errorf("%v", err)
			}

			httputils.HandleError(s, ctx, asErr)
		}
	}()

	ctx.Next()
}
