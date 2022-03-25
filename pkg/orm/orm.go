// orm package implements all database operations.
//
// Copyright 2022 Dr. Abiira Nathan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package orm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/abiiranathan/gosqlorm/pkg/query"
	"github.com/abiiranathan/gosqlorm/pkg/schema"
	"github.com/jackc/pgx/v4/pgxpool"
)

// String for the connection dialect
// Can be postgres,mysql or sqlite
// However only postgres is supported at the moment
type DriverName string

func (d DriverName) String() string {
	return string(d)
}

const (
	POSTGRES DriverName = "postgres"
	MYSQL    DriverName = "mysql"
	SQLITE   DriverName = "sqlite"
)

var (
	ErrInvalidDriver = errors.New("invalid driver")
	ErrDSNEmpty      = errors.New("dataSourceName is empty")
)

type Config struct {
	Driver         DriverName
	URI            string
	EnableFKChecks bool
	LoggerOutput   io.Writer
}

// GetDriver returns the driver name for the config c
func (c *Config) GetDriver() DriverName {
	return c.Driver
}

// Returns the dataSourceName used in the connection
func (c *Config) GetDSN() string {
	return c.URI
}

type ORM interface {
	// Find all records from the database for model
	FindAll(model interface{}, filter *query.QueryFilter) error

	// Find a single record from the database specified by the filter
	Find(model interface{}, filter *query.QueryFilter) error

	// Insert a new record v into the database
	Create(v interface{}) error

	// Update model v based on the consitions
	Update(v interface{}, conditions *query.QueryFilter) error

	// Delete model v based on conditions
	Delete(v interface{}, conditions *query.QueryFilter) error

	// Create all tables, constraints, relations for all models.
	// This is not a proper migration tool.
	//
	// TODO: Add proper migration magic for modifying schema
	AutoMigrate(models ...interface{}) error

	// Closes the connection pool
	Close()
}

// Concrete implementation for ORM interface
type orm struct {
	config *Config
	Pool   *pgxpool.Pool

	migrationErr error
}

// NewORM creates a new ORM instance using the config.
//
// If the config.Driver is an empty, it returns an error ErrInvalidDriver.
// If config.Driver != POSTGRES, returns an error.
func NewORM(config *Config) (ORM, error) {
	if config.Driver == "" {
		return nil, ErrInvalidDriver
	}

	if config.Driver != POSTGRES {
		return nil, fmt.Errorf("unsupported driver: %s. Only postgres is supported at the moment", config.Driver)
	}

	if config.URI == "" {
		return nil, ErrDSNEmpty
	}

	if config.LoggerOutput == nil {
		config.LoggerOutput = os.Stdout
	}

	pool, err := newDB(config)
	if err != nil {
		return nil, err
	}

	return &orm{
		config: config,
		Pool:   pool,
	}, nil
}

// connects to postgres database with config.URI
// If successful, it returns a pgxpool.Pool
// that is then attached to the ORM
//
// All queries use this connection pool
func newDB(config *Config) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(config.URI)
	if err != nil {
		return nil, err
	}

	conn, err := pgxpool.ConnectConfig(context.Background(), cfg)

	if err != nil {
		return nil, err
	}

	return conn, nil
}

// Return the configuration for the database
func (o *orm) GetConfig() *Config {
	return o.config
}

// Close closes all connections in the pool and rejects future Acquire calls.
//Blocks until all connections are returned to pool and closed.
func (o *orm) Close() {
	o.Pool.Close()
}

func (o *orm) FindAll(v interface{}, filter *query.QueryFilter) error {
	if !schema.IsPointerToArrayOfStructPointer(v) {
		return errors.New("v must be a pointer to a slice of structs")
	}

	model := schema.NewStructPointer(v)
	tableName := schema.GetTableName(model)
	_, qualified, _ := schema.Columns(model, o.config.Driver.String())

	buff := bytes.Buffer{}
	selector := strings.Trim(strings.Join(qualified, ", "), ", ")
	buff.WriteString(fmt.Sprintf("SELECT %s FROM %s ", selector, tableName))

	// Instantiate a new query object
	q := &query.Query{
		Driver: o.config.Driver.String(),
		Pool:   o.Pool,
		Query:  buff.String(),
		Result: v,
		Filter: filter,
	}

	return q.ScanAll()
}

// Find a single row in the table
// v should be a pointer to a struct
func (o *orm) Find(v interface{}, filter *query.QueryFilter) error {
	if !schema.IsStructPointer(v) {
		return errors.New("model v must be a pointer to a struct")
	}

	if err := filter.Validate(); err != nil {
		return err
	}

	model := schema.GetType(v)
	tableName := schema.GetTableName(model)
	_, qualified, _ := schema.Columns(model, o.config.Driver.String())

	buff := bytes.Buffer{}
	selector := strings.Trim(strings.Join(qualified, ", "), ", ")
	buff.WriteString(fmt.Sprintf("SELECT %s FROM %s ", selector, tableName))

	// Instantiate a new query object
	q := &query.Query{
		Driver: o.config.Driver.String(),
		Pool:   o.Pool,
		Query:  buff.String(),
		Result: v,
		Filter: filter,
	}

	return q.ScanOne()
}

// Insert a row into the table
func (o *orm) Create(v interface{}) error {
	if !schema.IsStructPointer(v) {
		return errors.New("model v must be a pointer to a struct")
	}

	insertQuery, values, err := schema.InsertSchema(v, o.config.Driver.String())
	if err != nil {
		return err
	}

	q := &query.Query{
		Driver: o.config.Driver.String(),
		Pool:   o.Pool,
		Query:  insertQuery,
		Result: v,
		Args:   values,
	}

	return q.Create()
}

// Updates model v based on specified conditions
func (o *orm) Update(v interface{}, conditions *query.QueryFilter) error {
	if !schema.IsStructPointer(v) {
		return errors.New("model v must be a pointer to a struct")
	}

	if err := conditions.Validate(); err != nil {
		return err
	}

	updateQuery, values, err := schema.UpdateSchema(v, conditions, o.config.Driver.String())
	if err != nil {
		return err
	}

	q := &query.Query{
		Driver: o.config.Driver.String(),
		Pool:   o.Pool,
		Query:  updateQuery,
		Result: v,
		Args:   values,
		Filter: conditions,
	}
	return q.Create()
}

// Deletes model v based on specified conditions
func (o *orm) Delete(v interface{}, conditions *query.QueryFilter) error {
	if !schema.IsStructPointer(v) {
		return errors.New("model v must be a pointer to a struct")
	}

	if err := conditions.Validate(); err != nil {
		return err
	}

	deleteQuery, err := schema.DeleteSchema(v, o.config.Driver.String())
	if err != nil {
		return err
	}

	q := &query.Query{
		Driver: o.config.Driver.String(),
		Pool:   o.Pool,
		Query:  deleteQuery,
		Result: v,
		Filter: conditions,
	}

	return q.Exec()
}

// Create all tables and relations.
//
// NB: This is not a migration tool. It's just a helper for creating all
// tables, their constraints, and relations.
func (o *orm) AutoMigrate(models ...interface{}) error {
	return schema.AutoMigrate(o.Pool, o.config.Driver.String(), models...)
}
