package builder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/query"
	"github.com/YuukanOO/seelf/pkg/storage"
)

const (
	perPage     = 5          // Number of elements per page. In the future, this may be configurable.
	countClause = "COUNT(*)" // Count clause needed for pagination.
)

var ErrPaginationNotSupported = errors.New("pagination not supported for this query. Did you forget to build the query using the Select function?")

type (
	// Tiny shortcut to a map of field name => field value
	Values map[string]any

	// Element which could execute queries on a database (direct connection or current transaction).
	Executor interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}

	// Statement function to append an SQL statement to a builder.
	Statement func(sqlBuilder)

	// Interface used to express the ability to append a SQL statement and arguments to a query
	// builder. This is a private interface to not pollute the public API.
	sqlBuilder interface {
		apply(string, ...any)
	}

	// Query builder result used to interact with the database.
	QueryBuilder[T any] interface {
		// F for format, append a raw SQL statement to the builder with the optional arguments.
		F(string, ...any) QueryBuilder[T]
		// S for Statement, apply one or multiple statements to this builder.
		S(...Statement) QueryBuilder[T]

		// Returns the SQL query generated
		String() string

		// Executes the query and returns all results
		All(Executor, context.Context, storage.Mapper[T], ...Dataloader[T]) ([]T, error)
		// Executes the query and returns the first matching result
		One(Executor, context.Context, storage.Mapper[T], ...Dataloader[T]) (T, error)
		// Returns a paginated data result set.
		Paginate(Executor, context.Context, storage.Mapper[T], int, ...Dataloader[T]) (query.Paginated[T], error)

		// Same as One but extract a primitive value by using a simple generic scanner
		Extract(Executor, context.Context) (T, error)
		// Executes the query without scanning the result.
		Exec(Executor, context.Context) error
	}
)

type queryBuilder[T any] struct {
	supportPagination bool
	parts             []string
	arguments         []any
}

// Builds up a new query.
func Query[T any](sql string, args ...any) QueryBuilder[T] {
	return newQuery[T](false, sql, args...)
}

// Starts a SELECT query with pagination handling. Giving fields with this function will make it
// possible to retrieve the total element count when calling .Paginate. Without this, pagination could
// not work.
func Select[T any](fields ...string) QueryBuilder[T] {
	return newQuery[T](true, "SELECT").F(strings.Join(fields, ","))
}

// Builds a new query for an INSERT statement.
func Insert(table string, values Values) QueryBuilder[any] {
	var (
		b       strings.Builder
		size             = len(values)
		i                = 0
		columns []string = make([]string, size)
		args    []any    = make([]any, size)
	)

	for field, value := range values {
		columns[i] = field
		args[i] = value
		i++
	}

	b.WriteString(fmt.Sprintf("INSERT INTO %s (", table))
	b.WriteString(strings.Join(columns, ","))

	placeholders := strings.Repeat(",?", size)[1:] // Remove the first comma
	b.WriteString(fmt.Sprintf(") VALUES (%s)", placeholders))

	return Command(b.String(), args...)
}

// Builds a new query for an UPDATE statement.
func Update(table string, values Values) QueryBuilder[any] {
	var (
		b          strings.Builder
		size                = len(values)
		i                   = 0
		statements []string = make([]string, size)
		args       []any    = make([]any, size)
	)

	for field, value := range values {
		statements[i] = fmt.Sprintf("%s = ?", field)
		args[i] = value
		i++
	}

	b.WriteString(fmt.Sprintf("UPDATE %s SET ", table))
	b.WriteString(strings.Join(statements, ","))

	return Command(b.String(), args...)
}

// Builds a new query without specifying a return type. Useful for commands such as
// INSERT, UPDATE and DELETE where no results are expected.
func Command(sql string, args ...any) QueryBuilder[any] {
	return Query[any](sql, args...)
}

func (q *queryBuilder[T]) F(sql string, args ...any) QueryBuilder[T] {
	q.parts = append(q.parts, sql)
	q.arguments = append(q.arguments, args...)
	return q
}

func (q *queryBuilder[T]) S(statements ...Statement) QueryBuilder[T] {
	for _, stmt := range statements {
		stmt(q)
	}

	return q
}

func (q *queryBuilder[T]) apply(sql string, args ...any) { q.F(sql, args...) }

func (q *queryBuilder[T]) All(
	ex Executor,
	ctx context.Context,
	mapper storage.Mapper[T],
	loaders ...Dataloader[T],
) ([]T, error) {
	rows, err := ex.QueryContext(ctx, q.String(), q.arguments...)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	results := make([]T, 0)

	// Instantiates needed stuff for data loaders
	// FIXME: maybe split this in a dedicated function to avoid the cost when no loaders are given
	mappings := make([]KeysMapping, len(loaders))

	for i := range mappings {
		mappings[i] = make(KeysMapping)
	}

	for rows.Next() {
		row, err := mapper(rows)

		if err != nil {
			return nil, err
		}

		for i, loader := range loaders {
			mappings[i][loader.ExtractKey(row)] = len(results)
		}

		results = append(results, row)
	}

	if len(loaders) > 0 {
		kr := KeyedResult[T]{
			data: results,
		}

		for i, loader := range loaders {
			kr.indexByKeys = mappings[i]

			if err = loader.Fetch(ex, ctx, kr); err != nil {
				return nil, err
			}
		}
	}

	return results, nil
}

func (q *queryBuilder[T]) Paginate(
	ex Executor,
	ctx context.Context,
	mapper storage.Mapper[T],
	page int,
	loaders ...Dataloader[T],
) (query.Paginated[T], error) {
	// FIXME: Since the query is definitely mutated to paginate the result, it could not
	// be runned twice. Maybe I should pass the query by value instead, not sure about that.
	var (
		err    error
		result = query.Paginated[T]{
			Page:        page,
			IsFirstPage: page == 1,
			PerPage:     perPage,
		}
	)

	if !q.supportPagination {
		return result, ErrPaginationNotSupported
	}

	// Replace field names with the count clause to retrieve the total number of elements for the query
	fields := q.parts[1]
	q.parts[1] = countClause

	if err = ex.QueryRowContext(ctx, q.String(), q.arguments...).Scan(&result.Total); err != nil {
		return result, err
	}

	// Restore the original fields, append the limit/offset clause and run the query
	q.parts[1] = fields
	q.F("LIMIT ? OFFSET ?", perPage, (result.Page-1)*perPage)

	result.IsLastPage = result.Total <= result.Page*perPage
	result.Data, err = q.All(ex, ctx, mapper, loaders...)

	return result, err
}

func (q *queryBuilder[T]) One(
	ex Executor,
	ctx context.Context,
	mapper storage.Mapper[T],
	loaders ...Dataloader[T],
) (T, error) {
	row := ex.QueryRowContext(ctx, q.String(), q.arguments...)

	result, err := mapper(row)

	if errors.Is(err, sql.ErrNoRows) {
		return result, apperr.ErrNotFound
	}

	if len(loaders) > 0 {
		kr := KeyedResult[T]{
			data: []T{result},
		}

		for _, loader := range loaders {
			kr.indexByKeys = KeysMapping{
				loader.ExtractKey(result): 0,
			}

			if err = loader.Fetch(ex, ctx, kr); err != nil {
				return result, err
			}
		}
	}

	return result, err
}

func (q *queryBuilder[T]) Extract(ex Executor, ctx context.Context) (T, error) {
	return q.One(ex, ctx, extract[T])
}

func (q *queryBuilder[T]) Exec(ex Executor, ctx context.Context) error {
	_, err := ex.ExecContext(ctx, q.String(), q.arguments...)

	return err
}

func (q *queryBuilder[T]) String() string {
	return strings.Join(q.parts, " ")
}

// Builds a new query with the given SQL and arguments. Also sets the supportPagination flag.
func newQuery[T any](supportPagination bool, sql string, args ...any) QueryBuilder[T] {
	return &queryBuilder[T]{
		supportPagination: supportPagination,
		parts:             []string{sql},
		arguments:         args,
	}
}
