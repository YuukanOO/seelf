package serve

import (
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/http"
	"github.com/gin-gonic/gin"
)

type listJobsFilters struct {
	Page int `form:"page"`
}

func (s *server) listJobsHandler() gin.HandlerFunc {
	return http.Bind(s, func(ctx *gin.Context, request listJobsFilters) error {
		var filters bus.GetJobsFilters

		if request.Page != 0 {
			filters.Page.Set(request.Page)
		}

		jobs, err := s.scheduledJobsStore.GetAllJobs(ctx.Request.Context(), filters)

		if err != nil {
			return err
		}

		return http.Ok(ctx, jobs)
	})
}

func (s *server) deleteJobsHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		err := s.scheduledJobsStore.Delete(ctx.Request.Context(), ctx.Param("id"))

		if err != nil {
			return err
		}

		return http.NoContent(ctx)
	})
}
