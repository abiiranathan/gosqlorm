package query

import (
	"context"
	"errors"
	"fmt"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	ErrInvalidDriver         = errors.New("invalid driver")
	ErrConnEmpty             = errors.New("connection is empty")
	ErrQueryEmpty            = errors.New("query is empty")
	ErrResultEmpty           = errors.New("result is nil")
	ErrEmptyQueryFilter      = errors.New("query filter cannot be nil")
	ErrEmptyQueryFilterWhere = errors.New("query filter where clause cannot be empty")
	ErrEmptyQueryFilterArgs  = errors.New("query filter args cannot be empty")
)

// Args is an alias for a slice of empty interface
type Args []interface{}

// Encapsulates a pgxpool.Pool and runs queries
type Query struct {
	// The database driver
	Driver string
	// The database connection string
	Pool *pgxpool.Pool

	// The query string
	Query string

	// QueryFileter
	Filter *QueryFilter

	// The query arguments
	Args Args

	// The query result
	Result interface{}

	// The query error
	Error error

	// The query context
	Context context.Context
}

// QueryFilters stores query filter clause with arguments to
// populate the query statement.
// Placeholders are written as : $1, $2, $3
type QueryFilter struct {
	// User defined raw query. Overrides the query.Query.Query field
	Query *string

	// Where condition
	Where string

	// Arguments for placeholders in Where clause. Must be equal
	Args Args

	// Keeps track of error while validating the query
	err error
}

// If the QueryFilter is nil, it returns ErrEmptyQueryFilter. If Where is empty, it returns ErrEmptyQueryFilterWhere.
// If len(qf.Args) ==0, it returns ErrEmptyQueryFilterArgs
func (qf *QueryFilter) Validate() error {
	if qf == nil {
		return ErrEmptyQueryFilter
	}

	if qf.err != nil {
		return qf.err
	}

	if qf.Where == "" {
		return ErrEmptyQueryFilterWhere
	}

	if len(qf.Args) == 0 {
		return ErrEmptyQueryFilterArgs
	}

	return nil
}

func (query *Query) AddQueryFilters() {
	if query.Filter == nil {
		return
	}

	if query.Filter.Query != nil {
		query.Query = *(query.Filter.Query)
	}

	if query.Filter.Where != "" && len(query.Filter.Args) > 0 {
		query.Query += " WHERE " + query.Filter.Where
		query.Args = append(query.Args, query.Filter.Args...)
	}

}

// Validates the query to make sure it has been instanciated with a good(not nil)
// Connection Pool, Query and Result struct.
// If the query context is nil, validate sets context.Background() on the query
func (q *Query) Validate() {
	if q.Pool == nil {
		q.Error = ErrConnEmpty
	}

	if q.Query == "" {
		q.Error = ErrQueryEmpty
	}

	if q.Result == nil {
		q.Error = ErrResultEmpty
	}

	if q.Context == nil {
		q.Context = context.Background()
	}
}

// Scans all rows in query Result
func (q *Query) ScanAll() error {
	q.Validate()

	if q.Error != nil {
		return q.Error
	}

	conn, err := q.Pool.Acquire(q.Context)
	if err != nil {
		return err
	}

	defer conn.Release()

	q.AddQueryFilters()

	fmt.Printf("[query] %s %v\n\n", q.Query, q.Args)
	return pgxscan.Select(q.Context, q.Pool, q.Result, q.Query, q.Args...)

}

// Scans a single row into the query result
func (q *Query) ScanOne() error {
	q.Validate()

	if q.Error != nil {
		return q.Error
	}

	conn, err := q.Pool.Acquire(q.Context)
	if err != nil {
		return err
	}

	defer conn.Release()

	q.AddQueryFilters()

	fmt.Printf("[query] %s %v\n\n", q.Query, q.Args)
	return pgxscan.Get(q.Context, q.Pool, q.Result, q.Query, q.Args...)
}

// Executes query q expecting no return values
func (q *Query) Exec() error {
	q.Validate()

	if q.Error != nil {
		return q.Error
	}

	q.AddQueryFilters()
	fmt.Printf("[query] %s %v\n\n", q.Query, q.Args)
	_, err := q.Pool.Exec(q.Context, q.Query, q.Args...)
	return err
}

// Executes the query and inserts new records into the database
func (q *Query) Create() error {
	q.Validate()

	if q.Error != nil {
		return q.Error
	}

	conn, err := q.Pool.Acquire(q.Context)
	if err != nil {
		return err
	}

	defer conn.Release()

	fmt.Printf("[query] %s: %v\n\n", q.Query, q.Args)
	// Exec does not return any rows
	err = pgxscan.Get(q.Context, q.Pool, q.Result, q.Query, q.Args...)
	return err
}
