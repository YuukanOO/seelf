package sqlite

import (
	"context"
	"embed"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
)

var (
	//go:embed migrations/*.sql
	migrations embed.FS

	Migrations = sqlite.NewMigrationsModule("scheduler", "migrations", migrations)
)

// Setup the needed infrastructure for the scheduler.
func Setup(
	b bus.Bus,
	db *sqlite.Database,
) (*JobsStore, error) {
	jobsStore := NewJobsStore(db, b)

	// Register some handlers to operate on jobs if needed.
	bus.Register(b, jobsStore.GetAllJobs)
	bus.Register(b, jobsStore.RetryJob)
	bus.Register(b, jobsStore.DismissJob)

	if err := db.Migrate(Migrations); err != nil {
		return nil, err
	}

	// And reset retrieved jobs to make sure they can be retried
	return jobsStore, jobsStore.ResetRetrievedJobs(context.Background())
}
