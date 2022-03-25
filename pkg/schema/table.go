package schema

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
)

type TableSchema struct {
	TableName        string
	Fields           []*Field
	PrimaryKey       *Field
	ForeignKeys      map[string]*ForeignKey
	UniqueFields     []*Field
	CompositeIndexes map[string][]*Field
	Constraints      []*Constraint

	buf      *bytes.Buffer
	migrated bool
}

type ForeignKey struct {
	ConstraintName string
	FK             string
	OnDelete       string
	OnUpdate       string
	TableName      string
	ParentTable    string
	ParentPkColumn string
}

type Constraint struct {
	Name  string
	Type  string
	Field *Field
}

var ForeignKeys = make(map[string][]*ForeignKey)

// Returns the sql string for creating the table
func (t *TableSchema) String(dialect string) string {
	if t.migrated {
		return t.buf.String()
	}

	t.buf = &bytes.Buffer{}
	t.WriteHeader()
	t.WriteColumns(dialect)
	t.WritePrimaryKey()
	t.WriteUniqueFields()
	t.WriteCompositeUnique()
	t.buf.WriteString("\n);")
	t.migrated = true
	return t.buf.String()
}

func (t *TableSchema) Flush() { t.buf.Reset() }

func (t *TableSchema) WriteHeader() {
	t.buf.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", t.TableName))

}

func (t *TableSchema) WriteColumns(dialect string) {
	for i, field := range t.Fields {
		if i > 0 && i <= len(t.Fields)-1 {
			if !field.IsForeignKey() {
				t.buf.WriteString(",\n")
			}
		}

		field.Table = t
		t.buf.WriteString(field.String())
	}
}

func (t *TableSchema) WritePrimaryKey() {
	if t.PrimaryKey != nil {
		t.buf.WriteString(fmt.Sprintf(",\nPRIMARY KEY (%s)", SnakeCase(t.PrimaryKey.Name)))
	}
}

func (t *TableSchema) WriteUniqueFields() {
	for _, field := range t.UniqueFields {
		t.buf.WriteString(fmt.Sprintf(",\nUNIQUE (%s)", SnakeCase(field.Name)))
	}
}

func (t *TableSchema) WriteCompositeUnique() {
	for _, fields := range t.CompositeIndexes {
		uniqueIndexes := []string{}
		for _, field := range fields {
			uniqueIndexes = append(uniqueIndexes, SnakeCase(field.Name))
		}

		t.buf.WriteString(fmt.Sprintf(",\nUNIQUE(%s)", strings.Join(uniqueIndexes, ", ")))
	}

}

// Returns the sql string for creating the table
func (table *TableSchema) InsertSchema(v interface{}, dialect string) (string, []interface{}) {
	buf := strings.Builder{}
	values := []interface{}{}
	buf.WriteString(fmt.Sprintf("INSERT INTO %s (", table.TableName))
	pkSkipped := false

	for i, field := range table.Fields {
		if field.IsForeignKey() {
			continue
		}

		if i > 0 {
			if !(pkSkipped && i == 1) {
				// If we have skipped the primary key, and we are on the second field,
				buf.WriteString(", ")
			}
		}

		refObjVal := reflect.ValueOf(v).Elem().FieldByName(field.Name)
		if field.IsPrimaryKey() && reflect.Zero(refObjVal.Type()).Interface() == refObjVal.Interface() {
			pkSkipped = true
			continue
		}

		buf.WriteString(SnakeCase(field.Name))
		values = append(values, refObjVal.Interface())
	}

	buf.WriteString(") VALUES (")

	// Loop through the fields and build the sql.
	// Initialize index (not loop index) to control i in the for loop
	i := 0
	for _, field := range table.Fields {
		if (field.IsPrimaryKey() && pkSkipped) || field.IsForeignKey() {
			continue
		}

		if i > 0 {
			buf.WriteString(", ")
		}

		buf.WriteString(fmt.Sprintf("$%d", i+1))
		i++

	}

	buf.WriteString(")")

	// Add returning clause
	if dialect == "postgres" {
		buf.WriteString(" RETURNING *")
	}

	return buf.String(), values
}

// Returns the sql string for updating the table
func (table *TableSchema) UpdateSchema(v interface{}, dialect string) (string, []interface{}) {
	buf := strings.Builder{}
	values := []interface{}{}
	buf.WriteString(fmt.Sprintf("UPDATE %s SET ", table.TableName))

	// Loop through the fields and build the sql.
	// Initialize index (not loop index) to control i in the for loop
	i := 0
	for loopIndex, field := range table.Fields {
		if field.IsPrimaryKey() || field.IsForeignKey() {
			continue
		}

		if i > 0 && loopIndex < len(table.Fields)-1 {
			buf.WriteString(", ")
		}

		buf.WriteString(fmt.Sprintf("%s = $%d", SnakeCase(field.Name), i+1))
		refObjVal := reflect.ValueOf(v).Elem().FieldByName(field.Name)
		values = append(values, refObjVal.Interface())
		i++
	}

	return buf.String(), values

}

// Returns the sql string for deleting the table with a trailing empty space
// Warning: Does not include the where clause
func (table *TableSchema) DeleteSchema(dialect string) string {
	return fmt.Sprintf("DELETE FROM %s ", table.TableName)
}

func (fk *ForeignKey) String() string {
	sql := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)",
		fk.TableName, fk.ConstraintName, SnakeCase(fk.FK), fk.ParentTable, SnakeCase(fk.ParentPkColumn))

	// Add fk contraints
	if fk.OnDelete != "" {
		sql += fk.OnDelete
	}

	if fk.OnUpdate != "" {
		sql += fk.OnUpdate
	}

	return sql
}
