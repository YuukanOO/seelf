package sqlite

import (
	"embed"

	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

var Migrations = sqlite.NewMigrationsModule("auth", "migrations", migrations)
