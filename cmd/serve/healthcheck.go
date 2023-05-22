package serve

import (
	"github.com/YuukanOO/seelf/pkg/http"
	"github.com/gin-gonic/gin"
)

type healthCheckResponse struct {
	Version string `json:"version"`
	Domain  string `json:"domain"`
}

func (s *server) healthcheckHandler(ctx *gin.Context) {
	http.Ok(ctx, healthCheckResponse{
		Version: s.options.CurrentVersion(),
		Domain:  s.options.Domain().String(),
	})
}
