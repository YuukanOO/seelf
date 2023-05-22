package builder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/storage"
)

const (
	// Number of elements per page. In the future, this may be configurable.
	perPage     = 5
	countClause = "COUNT(*)"
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

	// Scanner interface to handle entities with relations and to bulk queries.
	ScannerEx[T any, TConcreteScanner storage.Scanner] interface {
		storage.Scanner
		// Extend the given scanner to handle relationships.
		Contextualize(storage.Scanner) TConcreteScanner
		// Fetch related resources for the given entities.
		Finalize(context.Context, KeyedResult[T]) ([]T, error)
	}

	// Function to build a custom scanner for a given entity type.
	ScannerBuilder[T any, TScanner storage.Scanner] func(Executor) ScannerEx[T, TScanner]

	// Represents a key indexed set of data.
	KeyedResult[T any] struct {
		data        []T
		indexByKeys map[string]int
	}

	// Statement function to append an SQL statement to a builder.
	Statement func(sqlBuilder)

	// Interface used to express the ability to append a SQL statement and arguments to a query
	// builder. This is a private interface to not pollute the public API.
	sqlBuilder interface {
		apply(string, ...any)
	}

	// Query builder result used to interact with the database.
	QueryBuilder[T any, TScanner storage.Scanner] interface {
		// Append a raw SQL statement to the builder with the optional arguments.
		F(string, ...any) QueryBuilder[T, TScanner]
		// Apply one or multiple statements to this builder.
		S(...Statement) QueryBuilder[T, TScanner]

		// Returns the SQL query generated
		String() string
		// Executes the query and returns all results
		All(Executor, context.Context, storage.Mapper[T]) ([]T, error)
		// Returns a paginated data result set.
		Paginate(Executor, context.Context, storage.Mapper[T], int) (storage.Paginated[T], error)
		// Same as All but fetch related data using a custom scanner.
		AllEx(Executor, context.Context, ScannerBuilder[T, TScanner], storage.KeyedMapper[T, TScanner]) ([]T, error)
		// Executes the query and returns the first matching result
		One(Executor, context.Context, storage.Mapper[T]) (T, error)
		// Same as One but fetch related data using a custom scanner.
		OneEx(Executor, context.Context, ScannerBuilder[T, TScanner], storage.KeyedMapper[T, TScanner]) (T, error)
		// Same as One but extract a primitive value by using a simple generic scanner
		Extract(Executor, context.Context) (T, error)
		// Executes the query without scanning the result.
		Exec(Executor, context.Context) error

		// TODO: paginates and so on to make things simplier and avoid common mistakes (order of ? for examples)
	}
)

type query[T any, TScanner storage.Scanner] struct {
	supportPagination bool
	parts             []string
	arguments         []any
}

// Builds up a new query.
func Query[T any](sql string, args ...any) QueryBuilder[T, storage.Scanner] {
	return QueryEx[T, storage.Scanner](sql, args...)
}

// Builds up a new query with a custom scanner needed to retrieve relational data.
func QueryEx[T any, TScanner storage.Scanner](sql string, args ...any) QueryBuilder[T, TScanner] {
	return newQuery[T, TScanner](false, sql, args...)
}

// Starts a SELECT query with pagination handling. Giving fields with this function will make it
// possible to retrieve the total element count when calling .Paginate. Without this, pagination could
// not work.
func Select[T any](fields ...string) QueryBuilder[T, storage.Scanner] {
	return newQuery[T, storage.Scanner](true, "SELECT").F(strings.Join(fields, ","))
}

// Builds a new query for an INSERT statement.
func Insert(table string, values Values) QueryBuilder[any, storage.Scanner] {
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
func Update(table string, values Values) QueryBuilder[any, storage.Scanner] {
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
func Command(sql string, args ...any) QueryBuilder[any, storage.Scanner] {
	return Query[any](sql, args...)
}

func (q *query[T, TScanner]) F(sql string, args ...any) QueryBuilder[T, TScanner] {
	q.parts = append(q.parts, sql)
	q.arguments = append(q.arguments, args...)
	return q
}

func (q *query[T, TScanner]) S(statements ...Statement) QueryBuilder[T, TScanner] {
	for _, stmt := range statements {
		stmt(q)
	}

	return q
}

func (q *query[T, TScanner]) apply(sql string, args ...any) { q.F(sql, args...) }

func (q *query[T, TScanner]) All(ex Executor, ctx context.Context, mapper storage.Mapper[T]) ([]T, error) {
	rows, err := ex.QueryContext(ctx, q.String(), q.arguments...)

	if err != nil {
		return nil, err
	}

	results := make([]T, 0)

	defer rows.Close()

	for rows.Next() {
		row, err := mapper(rows)

		if err != nil {
			return nil, err
		}

		results = append(results, row)
	}

	return results, nil
}

func (q *query[T, TScanner]) Paginate(ex Executor, ctx context.Context, mapper storage.Mapper[T], page int) (storage.Paginated[T], error) {
	// FIXME: Since the query is definitely mutated to paginate the result, it could not
	// be runned twice. Maybe I should pass the query by value instead, not sure about that.
	var (
		err    error
		result = storage.Paginated[T]{
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

	if err := ex.QueryRowContext(ctx, q.String(), q.arguments...).Scan(&result.Total); err != nil {
		return result, err
	}

	// Restore the original fields, append the limit/offset clause and run the query
	q.parts[1] = fields
	q.F("LIMIT ? OFFSET ?", perPage, (result.Page-1)*perPage)

	result.IsLastPage = result.Total <= result.Page*perPage
	result.Data, err = q.All(ex, ctx, mapper)

	return result, err
}

func (q *query[T, TScanner]) AllEx(
	ex Executor,
	ctx context.Context,
	scannerBuilder ScannerBuilder[T, TScanner],
	mapper storage.KeyedMapper[T, TScanner],
) ([]T, error) {
	scanner := scannerBuilder(ex)
	rows, err := ex.QueryContext(ctx, q.String(), q.arguments...)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	results := KeyedResult[T]{
		data:        make([]T, 0),
		indexByKeys: make(map[string]int),
	}

	for rows.Next() {
		key, row, err := mapper(scanner.Contextualize(rows))

		if err != nil {
			return nil, err
		}

		results.indexByKeys[key] = len(results.data)
		results.data = append(results.data, row)
	}

	// If no results, just return the empty array, don't need to fetch related data
	if len(results.data) == 0 {
		return results.data, nil
	}

	return scanner.Finalize(ctx, results)
}

func (q *query[T, TScanner]) One(ex Executor, ctx context.Context, mapper storage.Mapper[T]) (T, error) {
	row := ex.QueryRowContext(ctx, q.String(), q.arguments...)

	result, err := mapper(row)

	if errors.Is(err, sql.ErrNoRows) {
		return result, apperr.ErrNotFound
	}

	return result, err
}

func (q *query[T, TScanner]) OneEx(
	ex Executor,
	ctx context.Context,
	scannerBuilder ScannerBuilder[T, TScanner],
	mapper storage.KeyedMapper[T, TScanner],
) (T, error) {

	scanner := scannerBuilder(ex)
	row := ex.QueryRowContext(ctx, q.String(), q.arguments...)

	key, result, err := mapper(scanner.Contextualize(row))

	if errors.Is(err, sql.ErrNoRows) {
		return result, apperr.ErrNotFound
	}

	if err != nil {
		return result, err
	}

	// FIXME: maybe create a finalizeOne which handle a single extended row
	rows, err := scanner.Finalize(ctx, KeyedResult[T]{
		data:        []T{result},
		indexByKeys: map[string]int{key: 0},
	})

	if err != nil {
		return result, err
	}

	return rows[0], nil
}

func (q *query[T, TScanner]) Extract(ex Executor, ctx context.Context) (T, error) {
	return q.One(ex, ctx, extract[T])
}

func (q *query[T, TScanner]) Exec(ex Executor, ctx context.Context) error {
	_, err := ex.ExecContext(ctx, q.String(), q.arguments...)

	return err
}

func (q *query[T, TScanner]) String() string {
	return strings.Join(q.parts, " ")
}

// Builds a new query with the given SQL and arguments. Also sets the supportPagination flag.
func newQuery[T any, TScanner storage.Scanner](supportPagination bool, sql string, args ...any) QueryBuilder[T, TScanner] {
	return &query[T, TScanner]{
		supportPagination: supportPagination,
		parts:             []string{sql},
		arguments:         args,
	}
}

func (r KeyedResult[T]) Data() []T { return r.data }

// Keys returns the list of keys contained in this dataset.
func (r KeyedResult[T]) Keys() []string {
	keys := make([]string, 0, len(r.indexByKeys))

	for key := range r.indexByKeys {
		keys = append(keys, key)
	}

	return keys
}
