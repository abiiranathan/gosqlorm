package main

import (
	"fmt"
	"log"

	"github.com/abiiranathan/gosqlorm/pkg/datatypes"
	"github.com/abiiranathan/gosqlorm/pkg/models"
	"github.com/abiiranathan/gosqlorm/pkg/orm"
	"github.com/abiiranathan/gosqlorm/pkg/query"
)

func main() {
	db, err := orm.NewORM(&orm.Config{
		Driver: "postgres",
		URI:    "postgres://nabiizy:password@localhost:5432/api?sslmode=disable",
	})

	if err != nil {
		log.Fatalf("error connecting to database: %v\n", err)
	}

	defer db.Close()

	err = db.AutoMigrate(
		&models.User{},
		&models.UserProfile{},
		&models.Contact{},
		&models.Token{},
	)

	if err != nil {
		log.Println(err)
	}

	_ = ListAll(db)
	_ = GetOne(db)
	user, err := Create(db)
	if err != nil {
		log.Printf("error creating record: %v", err)
		return
	}

	_ = UpdateUser(user, db)
	_ = Delete(user, db)

}

func ListAll(db orm.ORM) error {
	users := []*models.User{}

	err := db.FindAll(&users, nil)

	if err != nil {
		return err
	}

	for _, user := range users {
		fmt.Printf("ID: %d\n", user.ID)
		fmt.Printf("Name: %s\n", user.Name)
		fmt.Printf("Age: %d\n", user.Age)
		fmt.Printf("Details: %s\n", user.Details)
		fmt.Printf("BD: %s\n", user.BirthDate)
	}

	return nil
}

func GetOne(db orm.ORM) error {
	// Find a single row
	user := models.User{}
	err := db.Find(&user, &query.QueryFilter{Where: "id=$1", Args: query.Args{1}})

	if err != nil {
		return err
	}

	fmt.Printf("Name: %s\n", user.Name)
	return nil
}

func Create(db orm.ORM) (models.User, error) {
	// set bd to 1980-01-01
	var bd datatypes.Date
	bd, err := bd.FromString("1980-01-01")
	if err != nil {
		return models.User{}, err
	}

	u := models.User{
		Name:      "Kakura Nahason",
		Age:       45,
		BirthDate: bd,
		Details: map[string]interface{}{
			"username": "kakura",
			"password": "password",
		},
		Username: "kakura",
	}

	err = db.Create(&u)
	if err != nil {
		return models.User{}, err
	}

	fmt.Printf("New User ID: %d\n", u.ID)
	return u, nil
}

func UpdateUser(user models.User, db orm.ORM) error {
	// Update user
	user.Name = "Updated twice Kakura"
	err := db.Update(&user, &query.QueryFilter{Where: "id = $1 ", Args: query.Args{user.ID}})

	if err != nil {
		return fmt.Errorf("update Error: %v\n", err)
	}

	fmt.Printf("Updated Name: %s\n", user.Name)
	return nil
}

func Delete(user models.User, db orm.ORM) error {
	// Delete user
	err := db.Delete(&user, &query.QueryFilter{Where: "id = $1", Args: query.Args{user.ID}})
	if err != nil {
		return fmt.Errorf("unable to delete user: %v", err)
	}

	fmt.Printf("Deleted User ID: %d\n", user.ID)

	return nil
}
