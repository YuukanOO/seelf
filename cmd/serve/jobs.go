package serve

import (
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/embedded/dismiss_job"
	"github.com/YuukanOO/seelf/pkg/bus/embedded/get_jobs"
	"github.com/YuukanOO/seelf/pkg/bus/embedded/retry_job"
	"github.com/YuukanOO/seelf/pkg/http"
	"github.com/gin-gonic/gin"
)

type listJobsFilters struct {
	Page int `form:"page"`
}

func (s *server) listJobsHandler() gin.HandlerFunc {
	return http.Bind(s, func(ctx *gin.Context, request listJobsFilters) error {
		var filters get_jobs.Query

		if request.Page != 0 {
			filters.Page.Set(request.Page)
		}

		jobs, err := bus.Send(s.bus, ctx.Request.Context(), filters)

		if err != nil {
			return err
		}

		return http.Ok(ctx, jobs)
	})
}

func (s *server) dismissJobHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		if _, err := bus.Send(s.bus, ctx.Request.Context(), dismiss_job.Command{
			ID: ctx.Param("id"),
		}); err != nil {
			return err
		}

		return http.NoContent(ctx)
	})
}

func (s *server) retryJobHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		if _, err := bus.Send(s.bus, ctx.Request.Context(), retry_job.Command{
			ID: ctx.Param("id"),
		}); err != nil {
			return err
		}

		return http.NoContent(ctx)
	})
}
