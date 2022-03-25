package datatypes

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Custom date type to support dates of the format "2006-01-02".
//
// Create a new date instance with Date.FromString
type Date time.Time

// The date format
const layout = "2006-01-02"

// Satisfy database Scanner interface
func (date *Date) Scan(value interface{}) (err error) {
	nullTime := &sql.NullTime{}
	err = nullTime.Scan(value)
	*date = Date(nullTime.Time)
	return
}

// Satisfy database Valuer interface
func (date Date) Value() (driver.Value, error) {
	y, m, d := time.Time(date).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Time(date).Location()), nil
}

// GormDataType gorm common data type
func (date Date) GormDataType() string {
	return "date"
}

func (date Date) GobEncode() ([]byte, error) {
	return time.Time(date).GobEncode()
}

func (date *Date) GobDecode(b []byte) error {
	return (*time.Time)(date).GobDecode(b)
}

// Custom Json encoder
// Called when go types are being converted to json strings
func (date Date) MarshalJSON() ([]byte, error) {
	dateBytes, err := time.Time(date).MarshalJSON()
	if err != nil {
		return []byte(""), err
	}

	// Transform the date to format of layout in format yyyy-mm-dd
	dateString := string(dateBytes[1 : len(dateBytes)-1])
	dateString = fmt.Sprintf("\"%s-%02s-%02s\"", dateString[0:4], dateString[5:7], dateString[8:10])
	return []byte(dateString), nil

}

// Custom Json decoder
// Called to convert json strings to go types
func (date *Date) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("date should be a string, got %v", data)
	}

	// Make sure that the user has provided the standard date format
	_, err := time.Parse(layout, s)
	if err != nil {
		return fmt.Errorf("date should be of the format: yyyy-mm-dd")
	}

	// Convert date string to the standard format to RFC 3339 format
	s = fmt.Sprintf("\"%sT00:00:00Z\"", s)
	return (*time.Time)(date).UnmarshalJSON([]byte(s))
}

// Returns the year for date
func (date Date) Year() int {
	return time.Time(date).Year()
}

// Returns the month for date
func (date Date) Month() time.Month {
	return time.Time(date).Month()
}

// Returns the day for date
func (date Date) Day() int {
	return time.Time(date).Day()
}

// Stringer interface for date
// Of the format 2018-01-30
func (date Date) String() string {
	return fmt.Sprintf("%d-%02d-%02d", date.Year(), date.Month(), date.Day())
}

// Create a new date from year,month and day
// loc is the timezone location
func NewDate(year int, month time.Month, day int, loc *time.Location) Date {
	if loc == nil {
		loc = time.Local
	}

	return Date(time.Date(year, month, day, 0, 0, 0, 0, time.Local))
}

// FromString creates a new Date object from date string.
//
// If date is not of format matching layout: "2006-01-02", it returns an error
func (Date) FromString(date string) (Date, error) {
	t, err := time.Parse(layout, date)
	if err != nil {
		return Date{}, err
	}
	return Date(t), nil
}
