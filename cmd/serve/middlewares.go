package serve

import (
	"errors"
	"net/http"
	"strings"

	"github.com/YuukanOO/seelf/internal/auth/domain"
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
		sess := sessions.Default(ctx)
		uid, ok := sess.Get(userSessionKey).(string)
		failed := !ok || uid == ""

		// If it failed and api access is not allowed, return early
		if failed && !withApiAccess {
			ctx.AbortWithError(http.StatusUnauthorized, errUnauthorized)
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
			ctx.AbortWithError(http.StatusUnauthorized, errUnauthorized)
			return
		}

		id, err := s.usersReader.GetIDFromAPIKey(ctx.Request.Context(), domain.APIKey(authHeader[apiAuthPrefixLength:]))

		if err != nil {
			ctx.AbortWithError(http.StatusUnauthorized, errUnauthorized)
			return
		}

		// Attach the user id to the context passed down in every usecases.
		ctx.Request = ctx.Request.WithContext(domain.WithUserID(ctx.Request.Context(), id))

		ctx.Next()
	}
}

// Attach a database transaction to all requests that could possibly mutate data such
// as POST, PUT, PATCH, DELETE
func (s *server) transactional(ctx *gin.Context) {
	if !needsTransaction(ctx) {
		ctx.Next()
		return
	}

	s.logger.Debug("opening transaction")

	txContext, tx := s.db.WithTransaction(ctx.Request.Context())

	// FIXME: should I defer the commit here?

	ctx.Request = ctx.Request.WithContext(txContext)

	ctx.Next()

	if len(ctx.Errors) > 0 {
		s.logger.Debug("rollbacking transaction")

		if err := tx.Rollback(); err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}

	s.logger.Debug("commiting transaction")

	if err := tx.Commit(); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
	}
}

func needsTransaction(ctx *gin.Context) bool {
	switch ctx.Request.Method {
	case http.MethodHead,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodGet,
		http.MethodTrace:
		return false
	default:
		return true
	}
}
