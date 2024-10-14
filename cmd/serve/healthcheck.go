package serve

import (
	"github.com/YuukanOO/seelf/cmd/version"
	"github.com/YuukanOO/seelf/pkg/http"
	"github.com/gin-gonic/gin"
)

type healthCheckResponse struct {
	Version string `json:"version"`
}

func (s *server) healthcheckHandler(ctx *gin.Context) {
	_ = http.Ok(ctx, healthCheckResponse{
		Version: version.Current(),
	})
}
