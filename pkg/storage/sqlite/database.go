package sqlite

import (
	"context"
	"database/sql"
	"io/fs"
	"strings"
	"time"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/storage"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite/builder"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"
)

const (
	dbDriverName                     = "sqlite3"
	concurrencyColumnName            = "version"
	migrateSourceName                = "embed"
	transactionContextKey contextKey = "sqlitetx"
)

var (
	_ builder.Executor          = (*Database)(nil) // Ensure Database implements the Executor interface
	_ storage.UnitOfWorkFactory = (*Database)(nil)
)

type (
	// Represents a single module for database migrations.
	MigrationsModule struct {
		name string // Name of the module, used as a prefix for the migrations history table.
		dir  string // Relative directory in the fs containing *.sql migrations files.
		fs   fs.FS
	}

	// Handle to a sqlite database with useful helper methods on it :)
	Database struct {
		conn   *sql.DB
		bus    bus.Dispatcher
		logger log.Logger
	}

	contextKey string
)

// Opens a connection to a sqlite database file.
func Open(dsn string, logger log.Logger, bus bus.Dispatcher) (*Database, error) {
	db, err := sql.Open(dbDriverName, dsn)

	if err != nil {
		return nil, err
	}

	if err = db.PingContext(context.Background()); err != nil {
		return nil, err
	}

	return &Database{db, bus, logger}, nil
}

// Close the underlying database.
func (db *Database) Close() error {
	return db.conn.Close()
}

// Execute the given function in a transaction managing the commit and rollback
// based on the returned error if any.
func (db *Database) Create(ctx context.Context, fn func(context.Context) error) (finalErr error) {
	var (
		tx      *sql.Tx
		created bool
	)

	ctx, tx, created = db.WithTransaction(ctx)

	defer func() {
		if !created {
			return
		}

		var err error

		if finalErr != nil {
			err = tx.Rollback()
		} else {
			err = tx.Commit()
		}

		if err != nil {
			finalErr = err
		}
	}()

	finalErr = fn(ctx)
	return
}

// Migrates the opened database to the latest version.
func (db *Database) Migrate(modules ...MigrationsModule) error {
	for _, module := range modules {
		source, err := iofs.New(module.fs, module.dir)

		if err != nil {
			return err
		}

		driver, err := sqlite3.WithInstance(db.conn, &sqlite3.Config{
			MigrationsTable: module.name + "_" + sqlite3.DefaultMigrationsTable,
		})

		if err != nil {
			return err
		}

		migrator, err := migrate.NewWithInstance(migrateSourceName, source, dbDriverName, driver)

		if err != nil {
			return err
		}

		db.logger.Debugw("migrating database",
			"module", module.name)

		if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
			return err
		}
	}

	return nil
}

// Creates and enhance the given context with a transaction if no one exists yet.
// The returned boolean indicates if the transaction has been created by this call
// with true and if it returns false, it means the transaction has been initiated early.
//
// The caller MUST commit or rollback it.
//
// If the transaction could not be created, this method will panic.
func (db *Database) WithTransaction(ctx context.Context) (context.Context, *sql.Tx, bool) {
	tx := Transaction(ctx)

	if tx != nil {
		return ctx, tx, false
	}

	tx, err := db.conn.BeginTx(ctx, &sql.TxOptions{})

	if err != nil {
		panic(err)
	}

	return context.WithValue(ctx, transactionContextKey, tx), tx, true
}

// Retrieve the transaction in the given context if any, or nil if it doesn't
// have one.
func Transaction(ctx context.Context) *sql.Tx {
	val := ctx.Value(transactionContextKey)

	if val == nil {
		return nil
	}

	return val.(*sql.Tx)
}

func (db *Database) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.tryGetTransaction(ctx).ExecContext(ctx, query, args...)
}

func (db *Database) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.tryGetTransaction(ctx).QueryContext(ctx, query, args...)
}

func (db *Database) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return db.tryGetTransaction(ctx).QueryRowContext(ctx, query, args...)
}

// Retrieve the executor from the given context. This is needed to execute query
// in the current transaction if any could be found in the given context.
// If no transaction is opened, then the request is just sent to the connection.
func (db *Database) tryGetTransaction(ctx context.Context) builder.Executor {
	var querier builder.Executor = db.conn

	// FIXME: may be we should use prepared statement instead and have something along the line
	// func ApplyContextTx(ctx context.Context, stmt *sql.Stmt) *sql.Stmt {
	// 	if tx, ok := ctx.Value(txContextKey).(*sql.Tx); ok {
	// 		return tx.Stmt(stmt)
	// 	}
	// 	return stmt
	// }

	if tx := Transaction(ctx); tx != nil {
		querier = tx
	}

	return querier
}

type WriteMode uint8

const (
	WriteModeUpsert WriteMode = iota
	WriteModeDelete
)

type Key map[string]any

func (k Key) toSQL() (string, []any) {
	var b strings.Builder
	b.WriteString("WHERE TRUE")
	values := make([]any, 0, len(k))

	for n, v := range k {
		b.WriteString(" AND " + n + " = ?")
		values = append(values, v)
	}

	return b.String(), values
}

// Helpers to handle database writes from an array of event sources and handle events dispatching.
// It will open and manage a transaction if none exist in the given context. This way,
// we make sure event handlers participates in the same transaction so they are resolved as
// a whole.
//
// It will collect any field updates and then apply them altogether. It will also handle the concurrency
// version, adding it to the where clause and field values to make sure no concurrency is possible.
//
// There's no way to add this method to the DB without type conversion so this is the easiest way
// for now. Without the generics, I will always have to convert an array of entities to []event.Source
// which is not very convenient.
func WriteEvents[T event.Source](
	db *Database,
	ctx context.Context,
	entities []T,
	tableName string,
	key func(T) Key,
	collect func(event.Event, builder.Values) WriteMode,
) error {
	return db.Create(ctx, func(ctx context.Context) error {
		for _, ent := range entities {
			version, events := event.Unwrap(ent)

			// Skip empty entities
			if len(events) == 0 {
				continue
			}

			notifications := make([]bus.Signal, len(events)) // It's a shame Go could not accept an array of events as a slice of signals since Event are effectively Signal

			var (
				mode   = WriteModeUpsert
				values = builder.Values{}
			)

			// Collect updated columns and their values by looping through the events
			for i, evt := range events {
				m := collect(evt, values)

				// Delete mode should take precedence.
				// Maybe a Delete mode should break the loop early?
				if m != WriteModeUpsert {
					mode = m
				}

				notifications[i] = evt
			}

			var (
				nextVersion = time.Now().UTC()
				insert      = version.IsZero()
				whereClause string
				whereArgs   []any
			)

			// Append the next concurrency value to the bag of values
			values[concurrencyColumnName] = nextVersion

			// Build the WHERE clause based on entity primary key and current version,
			// only needed for UPDATE and DELETE statements
			if !insert {
				k := key(ent)
				k[concurrencyColumnName] = version
				whereClause, whereArgs = k.toSQL()
			}

			var b builder.QueryBuilder[any]

			switch mode {
			case WriteModeUpsert:
				if insert {
					b = builder.Insert(tableName, values)
				} else {
					b = builder.
						Update(tableName, values).
						F(whereClause, whereArgs...)
				}
			case WriteModeDelete:
				b = builder.
					Command("DELETE FROM "+tableName).
					F(whereClause, whereArgs...)
			}

			if err := b.MustExec(db, ctx); err != nil {
				return err
			}

			// Events has been processed, hydrate the entity with the new version number
			event.Hydrate(ent, nextVersion)

			if err := db.bus.Notify(ctx, notifications...); err != nil {
				return err
			}
		}

		return nil
	})
}

// Builds a new migrations module with the given module name (used as a migrations history table name prefix)
// and the directory where migrations are stored in the given filesystem.
func NewMigrationsModule(name string, dir string, fs fs.FS) MigrationsModule {
	return MigrationsModule{name, dir, fs}
}
