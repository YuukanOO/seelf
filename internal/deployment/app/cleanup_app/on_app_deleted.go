package cleanup_app

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

func OnAppDeletedHandler(artifactManager domain.ArtifactManager) bus.SignalHandler[domain.AppDeleted] {
	return func(ctx context.Context, evt domain.AppDeleted) error {
		return artifactManager.Cleanup(ctx, evt.ID)
	}
}
