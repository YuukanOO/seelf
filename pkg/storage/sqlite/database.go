package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"

	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite/builder"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"
)

const (
	migrationsDir                    = "migrations"
	dbDriverName                     = "sqlite3"
	migrateSourceName                = "embed"
	transactionContextKey contextKey = "sqlitetx"
)

type (
	Database interface {
		builder.Executor
		Dispatcher() event.Dispatcher
		WithTransaction(ctx context.Context) (context.Context, *sql.Tx)
		Migrate(MigrationsDir) error
		Close() error
	}

	MigrationsDir map[string]fs.FS

	// Handle to a sqlite database with useful helper methods on it :)
	database struct {
		conn       *sql.DB
		dispatcher event.Dispatcher
		logger     log.Logger
	}

	contextKey string
)

// Opens a connection to a sqlite database file.
func Open(dsn string, logger log.Logger, dispatcher event.Dispatcher) (Database, error) {
	db, err := sql.Open(dbDriverName, dsn)

	if err != nil {
		return nil, err
	}

	if err = db.PingContext(context.Background()); err != nil {
		return nil, err
	}

	return &database{db, dispatcher, logger}, nil
}

// Close the underlying database.
func (db *database) Close() error {
	return db.conn.Close()
}

// Migrates the opened database to the latest version.
func (db *database) Migrate(migrations MigrationsDir) error {
	for name, dir := range migrations {
		source, err := iofs.New(dir, migrationsDir)

		if err != nil {
			return err
		}

		driver, err := sqlite3.WithInstance(db.conn, &sqlite3.Config{
			MigrationsTable: fmt.Sprintf("%s_%s", name, sqlite3.DefaultMigrationsTable),
		})

		if err != nil {
			return err
		}

		migrator, err := migrate.NewWithInstance(migrateSourceName, source, dbDriverName, driver)

		if err != nil {
			return err
		}

		db.logger.Debugw("migrating database as needed", "context", name)

		if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
			return err
		}
	}

	return nil
}

// Creates and attach a transaction to the given context. It will be instantiated by
// a middleware so every writes will share the same tx, retrieved from the context.
// It will panic if transaction creation failed. The caller is responsible to
// commit / rollback the returned transaction.
func (db *database) WithTransaction(ctx context.Context) (context.Context, *sql.Tx) {
	tx, err := db.conn.BeginTx(ctx, &sql.TxOptions{})

	if err != nil {
		panic(err)
	}

	return context.WithValue(ctx, transactionContextKey, tx), tx
}

func (db *database) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.tryGetTransaction(ctx).ExecContext(ctx, query, args...)
}

func (db *database) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.tryGetTransaction(ctx).QueryContext(ctx, query, args...)
}

func (db *database) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return db.tryGetTransaction(ctx).QueryRowContext(ctx, query, args...)
}

// Retrieve the executor from the given context. This is needed to execute query
// in the current transaction if any could be found in the given context.
// If no transaction is opened, then the request is just sent to the connection.
func (db *database) tryGetTransaction(ctx context.Context) builder.Executor {
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

func (db *database) Dispatcher() event.Dispatcher { return db.dispatcher }

// Retrieve the transaction in the given context if any, or nil if it doesn't
// have one.
func Transaction(ctx context.Context) *sql.Tx {
	val := ctx.Value(transactionContextKey)

	if val == nil {
		return nil
	}

	return val.(*sql.Tx)
}

// Helpers to handle database writes from an array of event sources and handle events dispatching.
// It will open and manage a transaction if none exist in the given context. This way,
// we make sure event handlers participates in the same transaction so they are resolved as
// a whole.
//
// There's no way to add this method to the DB without type conversion so this is the easiest way
// for now.
func WriteAndDispatch[T event.Source](
	db Database,
	ctx context.Context,
	entities []T,
	switcher func(context.Context, event.Event) error,
) (finalErr error) {
	var (
		selfManaged bool
		tx          = Transaction(ctx)
	)

	// No transaction exists, let's managed it ourselves to make sure event handlers share
	// the same one
	if tx == nil {
		ctx, tx = db.WithTransaction(ctx)
		selfManaged = true
	}

	defer func() {
		if !selfManaged {
			return
		}

		if finalErr != nil {
			if err := tx.Rollback(); err != nil {
				finalErr = err
			}
		} else {
			finalErr = tx.Commit()
		}
	}()

	for _, ent := range entities {
		events := event.Unwrap(ent)

		for _, evt := range events {
			if finalErr = switcher(ctx, evt); finalErr != nil {
				return
			}
		}

		if finalErr = db.Dispatcher().Dispatch(ctx, events...); finalErr != nil {
			return
		}

		// TODO: clear entities events (see #71)
	}

	return nil
}
