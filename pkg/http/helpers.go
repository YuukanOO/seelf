package http

import (
	"errors"
	"fmt"
	"net/http"
	"path"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/gin-gonic/gin"
)

var ErrUnexpected = apperr.New("unexpected_error") // Error returned when an infrastructure error occurs

// Tiny interface to represents needed contrat in order to use helpers provided by this package.
type Server interface {
	IsSecure() bool
	Logger() log.Logger
}

// Bind the request to the given TIn and handle errors if any by setting the appropriate status.
func Bind[TIn any](s Server, handler func(*gin.Context, TIn) error) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var cmd TIn

		if err := ctx.ShouldBind(&cmd); err != nil {
			ctx.AbortWithError(http.StatusUnprocessableEntity, err)
			return
		}

		if err := handler(ctx, cmd); err != nil {
			HandleError(s, ctx, err)
		}
	}
}

// Call the given handler and handle any returned error by setting the appropriate status.
func Send(s Server, handler func(*gin.Context) error) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if err := handler(ctx); err != nil {
			HandleError(s, ctx, err)
		}
	}
}

// Sets the location header and the response status
func Created[TOut any](s Server, ctx *gin.Context, data TOut, location string, args ...any) error {
	scheme := "http://"

	if s.IsSecure() {
		scheme = "https://"
	}

	addCommonResponseHeaders(ctx)
	ctx.Header("Location", fmt.Sprintf(scheme+path.Join(ctx.Request.Host, location), args...))
	ctx.JSON(http.StatusCreated, data)
	return nil
}

// Mark the request has succeeded with no data.
func NoContent(ctx *gin.Context) error {
	addCommonResponseHeaders(ctx)
	ctx.Status(http.StatusNoContent)
	return nil
}

// Returns the file at the given path.
func File(ctx *gin.Context, filepath string) error {
	ctx.File(filepath)
	return nil
}

// Mark the request has succeeded with the given data.
func Ok[TOut any](ctx *gin.Context, data TOut) error {
	addCommonResponseHeaders(ctx)
	ctx.JSON(http.StatusOK, data)
	return nil
}

// Handle the given non-nil error and sets the status code based on error type.
func HandleError(s Server, ctx *gin.Context, err error) {
	var (
		status int = http.StatusInternalServerError
		data   any = ErrUnexpected
	)

	// Translates the error type to the appropriate HTTP status code
	if _, isAppErr := apperr.As[apperr.Error](err); isAppErr {
		status = http.StatusBadRequest // Default to HTTP 400
		data = err

		if errors.Is(err, apperr.ErrNotFound) {
			status = http.StatusNotFound // But if it's a not found, that's an HTTP 404
		}
	} else {
		s.Logger().Errorw(err.Error(), "error", err)
	}

	ctx.Error(err)
	ctx.AbortWithStatusJSON(status, data)
}

func addCommonResponseHeaders(ctx *gin.Context) {
	ctx.Header("Cache-Control", "public, max-age=0, must-revalidate")
}
