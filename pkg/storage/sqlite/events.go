package sqlite

import (
	"context"
	"strings"
	"time"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite/builder"
)

type WriteMode uint8

const (
	WriteModeUpsert WriteMode = iota
	// WriteModeInsert
	// WriteModeUpdate
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

func WriteEventsBis[T event.Source](
	db *Database,
	ctx context.Context,
	entities []T,
	tableName string,
	key func(T) Key,
	collect func(event.Event, *Context),
	options ...WriteEventsOptionBuilder,
) error {
	opts := &writeEventsOptions{}

	for _, f := range options {
		f(opts)
	}

	return db.Create(ctx, func(ctx context.Context) error {
		for _, ent := range entities {
			version, events := event.Unwrap(ent)

			// Skip empty entities
			if len(events) == 0 {
				continue
			}

			notifications := make([]bus.Signal, len(events)) // It's a shame Go could not accept an array of events as a slice of signals since Event are effectively Signal

			context := &Context{
				Values: builder.Values{},
			}

			// Collect updated columns and their values by looping through the events
			for i, evt := range events {
				collect(evt, context)

				notifications[i] = evt
			}

			var (
				nextVersion = time.Now().UTC()
				insert      = version.IsZero()
				whereClause string
				whereArgs   []any
			)

			// Append the next concurrency value to the bag of values
			context.Values[concurrencyColumnName] = nextVersion

			// Build the WHERE clause based on entity primary key and current version,
			// only needed for UPDATE and DELETE statements
			if !insert {
				k := key(ent)
				k[concurrencyColumnName] = version
				whereClause, whereArgs = k.toSQL()
			}

			var b builder.QueryBuilder[any]

			switch context.mode {
			case WriteModeUpsert:
				if insert {
					b = builder.Insert(tableName, context.Values)

					if onConflict, hasOnConflict := opts.onConflict.TryGet(); hasOnConflict {
						b.S(builder.OnConflictDoUpdate(onConflict.column, context.Values, onConflict.ignore...))
					}
				} else {
					b = builder.
						Update(tableName, context.Values).
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

type (
	onConflict struct {
		column string
		ignore []string
	}

	Context struct {
		mode WriteMode

		Values builder.Values // Values to be written for the aggregate being persisted
	}

	writeEventsOptions struct {
		onConflict monad.Maybe[onConflict]
	}

	WriteEventsOptionBuilder func(wo *writeEventsOptions)
)

// Mark the context as a delete operation
func (c *Context) Delete() {
	c.mode = WriteModeDelete
}

// Register an on conflict do update clause which will update values when a conflict occurs on the
// given column. Columns names in the valuesToIgnore slice will be ignored from the context
// values when updating.
func UpdateOnConflict(column string, valuesToIgnore ...string) WriteEventsOptionBuilder {
	return func(wo *writeEventsOptions) {
		wo.onConflict.Set(onConflict{
			column: column,
			ignore: valuesToIgnore,
		})
	}
}
