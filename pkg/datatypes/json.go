// The datatype package implements custom data types like Date and JSON.
//
// Copyright 2022 Dr. Abiira Nathan. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package datatypes

import (
	"database/sql/driver"
	"encoding/json"
)

// Custom JSON data type that implements the sql.Scanner and driver.Valuer interfaces
// to work with postgres database.
type JSON map[string]interface{}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	if err := json.Unmarshal(value.([]byte), &j); err != nil {
		return err
	}
	return nil
}

// Value returns the json value
//
// Implement driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	valueString, err := json.Marshal(j)
	return string(valueString), err
}
