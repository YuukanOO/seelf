package serve

import (
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/embedded/get_jobs"
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
