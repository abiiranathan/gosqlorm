package schema

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/abiiranathan/gosqlorm/pkg/query"
	"github.com/jackc/pgx/v4/pgxpool"
)

// Returns true if tag in tags
func hasTag(tag string, tags []string) bool {
	for _, t := range tags {
		if strings.Contains(t, tag) {
			return true
		}
	}
	return false
}

//  Generates the table schema for struct m
//
// Returns a pointer to TableSchema and an error if m is not a struct
// or pointer to a struct
func GetTableSchema(m interface{}, dialect string) (*TableSchema, error) {
	tblSchema := &TableSchema{}
	tblSchema.CompositeIndexes = make(map[string][]*Field)
	tblSchema.ForeignKeys = make(map[string]*ForeignKey)
	var v = m

	if IsPointer(v) {
		v = reflect.ValueOf(v).Elem().Interface()
	}

	if !IsStruct(v) {
		return nil, fmt.Errorf("%s is not a struct", reflect.TypeOf(v).Name())
	}

	tblSchema.TableName = SnakeCase(reflect.TypeOf(v).Name())
	tblSchema.Fields = make([]*Field, 0)
	tblSchema.Constraints = make([]*Constraint, 0)

	for i := 0; i < reflect.TypeOf(v).NumField(); i++ {
		field := reflect.TypeOf(v).Field(i)
		fieldValue := reflect.ValueOf(v).Field(i)

		if field.PkgPath != "" {
			continue
		}

		// Construct field with its tags using reflection
		fieldSchema := &Field{
			Name:            field.Name,
			Type:            field.Type.String(),
			ReflectObjType:  &field,
			ReflectObjValue: &fieldValue,
			buf:             &bytes.Buffer{},
			dialect:         dialect,
		}

		// Keep tags in a map
		fieldSchema.Tags = make(map[string]string)

		// ORM tags
		tags := strings.Split(field.Tag.Get("orm"), ";")

		for _, tag := range tags {
			tag = strings.TrimSpace(tag)

			if tag == "" {
				continue
			}

			tagParts := strings.Split(tag, ":")
			if len(tagParts) == 2 {
				tagName := strings.TrimSpace(tagParts[0])
				fieldSchema.Tags[tagName] = strings.TrimSpace(tagParts[1])
			} else {
				fieldSchema.Tags[tagParts[0]] = ""
			}
		}

		tblSchema.Fields = append(tblSchema.Fields, fieldSchema)
	}

	tblSchema.TableName = GetTableName(v)

	return tblSchema, nil

}

// Calls GetTableSchema to generate the sql for creating the table
// and all constraints. If you are interested in the TableSchema data structure
// call schema.GetTableSchema
func Schema(v interface{}, dialect string) (string, error) {
	tblSchema, err := GetTableSchema(v, dialect)
	if err != nil {
		return "", err
	}

	return tblSchema.String(dialect), nil
}

// Returns a slice table columns, qualified_column_names and an error
func Columns(v interface{}, dialect string) ([]string, []string, error) {
	tblSchema, err := GetTableSchema(v, dialect)
	if err != nil {
		return []string{}, []string{}, err
	}

	cols := tblSchema.Fields
	columns := make([]string, len(cols))
	qualifiedColumns := make([]string, len(cols))

	for i, col := range cols {
		if col.IsForeignKey() {
			continue
		}

		qualifiedColumns[i] = fmt.Sprintf("%s.%s", tblSchema.TableName, SnakeCase(col.Name))
		columns[i] = col.Name
	}

	return columns, qualifiedColumns, nil
}

// Returns the string for the Insert query
func InsertSchema(v interface{}, dialect string) (string, []interface{}, error) {
	tblSchema, err := GetTableSchema(v, dialect)
	if err != nil {
		return "", nil, err
	}

	insertString, values := tblSchema.InsertSchema(v, dialect)
	return insertString, values, nil
}

// Returns the string for the UpdateQuery
func UpdateSchema(v interface{}, filter *query.QueryFilter, dialect string) (string, []interface{}, error) {
	tblSchema, err := GetTableSchema(v, dialect)
	if err != nil {
		return "", nil, err
	}

	if err := filter.Validate(); err != nil {
		return "", nil, err
	}

	updateString, values := tblSchema.UpdateSchema(v, dialect)
	lastParam := len(values)
	updateString += " WHERE "

	whereClase := filter.Where

	// Append where clause placeholders to the query
	// There should probably be a better way to do this
	// but I'm not sure how to do it right now
	for i, v := range filter.Args {
		// Replace $in of the form $1,$2,$3 to start from lastParam
		if strings.Contains(whereClase, "$"+strconv.FormatInt(int64(i)+1, 10)) {
			whereClase = strings.Replace(whereClase, "$"+strconv.FormatInt(int64(i)+1, 10), "$"+strconv.FormatInt(int64(lastParam)+1, 10), -1)
			lastParam++
		}

		values = append(values, v)
	}

	updateString += whereClase

	// Add returning clause
	if dialect == "postgres" {
		updateString += " RETURNING *"
	}

	return updateString, values, nil
}

// Returns the string for DELETE statement
func DeleteSchema(v interface{}, dialect string) (string, error) {
	tblSchema, err := GetTableSchema(v, dialect)
	if err != nil {
		return "", err
	}
	return tblSchema.DeleteSchema(dialect), nil
}

// Creates all tables, constraints and relations.
// NB: This does not alter existing table schema and is not recommendated
// as a solid migration option.
func AutoMigrate(pool *pgxpool.Pool, driver string, models ...interface{}) error {
	schemasObjects := map[string]*TableSchema{}
	for _, model := range models {
		s, err := GetTableSchema(model, driver)
		if err != nil {
			return err
		}

		schemasObjects[s.TableName] = s

		// Populate the table table schema and foreign keys by calling String() method
		s.String(driver)
	}

	for tableName, tableSchema := range schemasObjects {
		// Create the table if it doesn't exist
		sql := tableSchema.String(driver)
		fmt.Println(sql)

		// Execute create table statement
		_, err := pool.Exec(context.Background(), sql)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error creating table %s: %v", tableName, err)
			continue
		}

		// If the tableName has no foreignKeys, go to the next table
		if _, ok := ForeignKeys[tableName]; !ok {
			continue
		}

		// Create the foreign keys for tableName
		for _, fk := range ForeignKeys[tableName] {
			sql := fk.String()
			fmt.Println(sql)
			_, err = pool.Exec(context.Background(), sql)

			if err != nil {
				if !strings.Contains(err.Error(), "already exists") {
					return err
				}
			}
		}
	}

	return nil
}
