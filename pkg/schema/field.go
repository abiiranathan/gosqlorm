package schema

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
)

// Field is the data structure that stores data about a single struct field
type Field struct {
	Name            string
	Type            string
	Tags            map[string]string
	ReflectObjType  *reflect.StructField
	ReflectObjValue *reflect.Value
	Table           *TableSchema
	buf             *bytes.Buffer
	dialect         string
}

func (f *Field) IsPrimaryKey() bool {
	isPk := false

	for tagName := range f.Tags {
		if tagName == "primaryKey" {
			isPk = true
			break
		}
	}

	return isPk
}

func (field *Field) IsForeignKey() bool {
	isFk := false

	for tagName := range field.Tags {
		if tagName == "foreignKey" {
			isFk = true
			break
		}
	}
	return isFk
}

func (f *Field) IsPrimaryKeyAndZero() bool {
	isPk := false

	for tagName := range f.Tags {
		if tagName == "primaryKey" {
			isPk = true
			break
		}
	}

	return isPk
}

func (f *Field) IsConstraint(tagName string) bool {
	flag := false
	for _, t := range []string{"unique", "check", "uniqueIndex", "autoIncrement", "foreignKey", "onDelete", "onUpdate"} {
		if tagName == t {
			flag = true
			break
		}
	}

	return flag
}

func (f *Field) IsAutoIncrement() bool {
	isAuto := false

	for tagName := range f.Tags {
		if tagName == "autoIncrement" {
			isAuto = true
			break
		}
	}

	return isAuto
}

// Checks if a foreign key with constraint constraint_name exists
// in a global map of foreign keys
func (f *Field) FKExists(constraint_name string) bool {
	exists := false
	for _, fksList := range ForeignKeys {
		for _, fk := range fksList {
			if fk.ConstraintName == constraint_name {
				exists = true
				break
			}
		}
	}
	return exists
}

// Write field tags representing constraints to the underlying field bytes.Buffer
func (f *Field) WriteFieldConstraints(k, v string) {
	if k == "unique" {
		f.Table.UniqueFields = append(f.Table.UniqueFields, f)
	} else if k == "autoIncrement" {
		f.buf.WriteString(" ")
		if f.dialect == "postgres" {
			// Skip auto increment for postgres, will use serial type
		} else if f.dialect == "mysql" {
			f.buf.WriteString("AUTO_INCREMENT")
		} else if f.dialect == "sqlite" {
			f.buf.WriteString("AUTOINCREMENT")
		}
	} else if k == "uniqueIndex" {
		f.Table.CompositeIndexes[v] = append(f.Table.CompositeIndexes[v], f)
	} else if k == "foreignKey" {
		// v should of the form fk->id
		fks := strings.Split(v, "->")

		if len(fks) != 2 {
			panic(fmt.Sprintf("Invalid foreign key definition: %s", v))
		}

		// Get struct type of the foreign key field
		fkStructType := f.ReflectObjValue.Interface()
		// Get the foreign key field
		// append to map
		constraint_name := fmt.Sprintf("%s_%s_fkey", SnakeCase(f.Table.TableName), SnakeCase(f.Name))
		if f.FKExists(constraint_name) {
			return
		}

		TableName := GetTableName(fkStructType)
		fk := &ForeignKey{
			ConstraintName: constraint_name,
			OnDelete:       "",
			OnUpdate:       "",
			FK:             fks[0],
			ParentPkColumn: fks[1],
			TableName:      TableName,
			ParentTable:    SnakeCase(f.Table.TableName),
		}

		ForeignKeys[TableName] = append(ForeignKeys[TableName], fk)

		// Get onDelete and onUpdate Constraints
		for k, v := range f.Tags {
			if k == "onDelete" {
				fk.OnDelete = fmt.Sprintf(" ON DELETE %s", v)
			} else if k == "onUpdate" {
				fk.OnUpdate = fmt.Sprintf(" ON UPDATE %s", v)
			}
		}

	} else if k == "check" {
		f.buf.WriteString(fmt.Sprintf(" CHECK (%s)", v))
	}
}

// Writes column name and type to the buffer
func (f *Field) PrintType(sqlType string, dialect string) {
	f.buf.WriteString("  " + SnakeCase(f.Name))
	f.buf.WriteString(" ")

	if f.IsAutoIncrement() {
		if dialect == "postgres" {
			f.buf.WriteString("SERIAL")
		} else {
			f.buf.WriteString(strings.ToUpper(sqlType))
		}
		return
	}

	if dialect == "postgres" && sqlType == "json" {
		f.buf.WriteString("JSONB")
		return
	}

	f.buf.WriteString(strings.ToUpper(sqlType))
}

// Print all field tags to the field buffer
func (f *Field) PrintTags() {
	for k, v := range f.Tags {
		if k == "type" || k == "primaryKey" {
			continue
		}

		if f.IsConstraint(k) {
			f.WriteFieldConstraints(k, v)
		} else {
			if v == "" {
				// No tag data, just print the key
				f.buf.WriteString(" ")
				f.buf.WriteString(k)
			} else {
				// Print the key and value
				f.buf.WriteString(" ")
				f.buf.WriteString(k)
				f.buf.WriteString(" ")
				f.buf.WriteString(v)
			}
		}
	}
}

// Returns the complete string representing the schema of a single column
//
// e.g : name varchar(200) not null unique
func (f *Field) String() string {
	if f.Tags["type"] != "" {
		f.PrintType(f.Tags["type"], f.dialect)
	} else {
		sqlType := OrmType(f.ReflectObjValue)

		if sqlType != "" {
			f.PrintType(sqlType, f.dialect)
		}
	}

	if f.IsPrimaryKey() {
		f.Table.PrimaryKey = f
	}

	f.PrintTags()

	return f.buf.String()
}
