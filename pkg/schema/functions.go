package schema

import (
	"reflect"
	"strings"
	"time"

	"github.com/abiiranathan/gosqlorm/pkg/datatypes"
	"github.com/google/uuid"
	"github.com/iancoleman/strcase"
	"github.com/lib/pq"
)

// convert strings to snake case without reflection
func SnakeCase(s string) string {
	return strcase.ToSnake(s)
}

// Returns true if v is a pointer
func IsPointer(v interface{}) bool {
	return reflect.ValueOf(v).Kind() == reflect.Pointer
}

// Returns true if v is a slice
func IsSlice(v interface{}) bool {
	return reflect.ValueOf(v).Kind() == reflect.Slice
}

// Returns true if v is a struct
func IsStruct(v interface{}) bool {
	return reflect.ValueOf(v).Kind() == reflect.Struct
}

// Returns true if v is a pointer to a struct
func IsStructPointer(v interface{}) bool {
	return IsPointer(v) && IsStruct(reflect.ValueOf(v).Elem().Interface())
}

// Creates a new struct pointer from a *[]*Model
func NewStructPointer(model interface{}) any {
	return reflect.New(reflect.TypeOf(model).Elem().Elem().Elem()).Interface()
}

// v must be a of the form *[]*Model{}
func IsPointerToArrayOfStructPointer(v interface{}) bool {
	return reflect.TypeOf(v).Kind() == reflect.Ptr &&
		reflect.TypeOf(v).Elem().Kind() == reflect.Slice &&
		reflect.TypeOf(v).Elem().Elem().Kind() == reflect.Ptr &&
		reflect.TypeOf(v).Elem().Elem().Elem().Kind() == reflect.Struct
}

func GetTableName(v interface{}) string {
	for i := 0; i < reflect.TypeOf(v).NumMethod(); i++ {
		method := reflect.TypeOf(v).Method(i)

		if method.PkgPath != "" {
			continue
		}

		if method.Name == "TableName" {
			return method.Func.Call([]reflect.Value{reflect.ValueOf(v)})[0].String()
		}

	}

	tblName := SnakeCase(reflect.TypeOf(v).Name())
	return pleuralize(tblName)
}

// if s ends with y -> ies
// s ends with s -> do not modify
// otherwise add s
func pleuralize(s string) string {
	if strings.HasSuffix(s, "y") {
		return s[0:len(s)-1] + "ies"
	}
	if strings.HasSuffix(s, "s") {
		return s
	}
	return s + "s"
}

// OrmType uses reflection to guess corresponding database type
func OrmType(v *reflect.Value) string {
	var sqlType string

	switch v.Kind() {
	case reflect.String:
		sqlType = "varchar(255)"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		sqlType = "integer"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		sqlType = "integer"
	case reflect.Float32, reflect.Float64:
		sqlType = "real"
	case reflect.Bool:
		sqlType = "boolean"
	case reflect.Array:
		sqlType = "array"
		if _, ok := v.Interface().(uuid.UUID); ok {
			sqlType = "uuid"
		}
	case reflect.Slice:
		if _, ok := v.Interface().(pq.StringArray); ok {
			sqlType = "text[]"
		} else if _, ok := v.Interface().(pq.Int64Array); ok {
			sqlType = "integer[]"
		} else if _, ok := v.Interface().(pq.Float64Array); ok {
			sqlType = "real[]"
		} else if _, ok := v.Interface().(pq.BoolArray); ok {
			sqlType = "boolean[]"
		} else if _, ok := v.Interface().(pq.ByteaArray); ok {
			sqlType = "bytea[]"
		} else if _, ok := v.Interface().(pq.StringArray); ok {
			sqlType = "text[]"
		} else {
			sqlType = "text[]"
		}
	case reflect.TypeOf(datatypes.JSON{}).Kind():
		sqlType = "json"
	case reflect.Struct:
		// If it's a time.Time, we'll assume it's a timestamp
		if v.Type() == reflect.TypeOf(datatypes.Date{}) {
			sqlType = "date"
		} else if v.Type() == reflect.TypeOf(time.Time{}) {
			sqlType = "timestamptz"
		}
	}

	return sqlType
}

// Initializes a pointer to the underlying model struct
// model must be a struct pointer
// e.g model := &Model{}
func GetType(model any) any {
	return reflect.New(reflect.TypeOf(model).Elem()).Interface()
}
