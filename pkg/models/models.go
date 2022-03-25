package models

import (
	"github.com/abiiranathan/gosqlorm/pkg/datatypes"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type UserProfile struct {
	ID     int    `json:"id" orm:"primaryKey"`
	Name   string `json:"name" orm:"not null" json:"name"`
	UserID int    `json:"user_id" orm:"not null;unique" binding:"required"`
}

type Contact struct {
	ID     int    `json:"id" orm:"primaryKey"`
	Mobile string `json:"mobile" gorm:"not null" `
	UserID int    `json:"user_id" orm:"not null;unique" binding:"required"`
}

type User struct {
	ID        int64          `json:"id" orm:"primaryKey;not null;autoIncrement"`
	Name      string         `json:"name" orm:"type:varchar(200);not null;uniqueIndex:username_index" validation:"required"`
	Age       int64          `json:"age" orm:"not null;default:20;check:age > 20" validation:"required"`
	BirthDate datatypes.Date `json:"birth_date" orm:"not null" validation:"required"`
	Details   datatypes.JSON `json:"details" orm:"type:jsonb"`
	Username  string         `json:"username" orm:"not null;uniqueIndex:username_index;check:age > 20" validation:"required"`

	Profile UserProfile `json:"profile" orm:"foreignKey:UserID->ID;onDelete:CASCADE; onUpdate:CASCADE"`
	Contact Contact     `json:"contact" orm:"foreignKey:UserID->ID;onDelete:CASCADE; onUpdate:CASCADE"`
	Token   Token       `json:"tokens" orm:"foreignKey:UserID->ID;onDelete:CASCADE; onUpdate:CASCADE"`
}

// Returns a strings representing the table name
func (u User) TableName() string {
	return "examples"
}

type Token struct {
	UUID       uuid.UUID      `json:"uuid" orm:"primaryKey;not null"`
	UserID     int            `json:"user_id" orm:"not null"`
	Privileges pq.StringArray `json:"privileges"`
}
