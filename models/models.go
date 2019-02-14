package models

import (
	/*blank import to make drivers available*/
	_ "database/sql"
	"fmt"
	/*blank import to make drivers available*/
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/utils"
	"gopkg.in/go-playground/validator.v8"
)

var (
	/*DB is our connection to the database*/
	DB *gorm.DB
	/*Validator will let us validate our structs*/
	Validator *validator.Validate
)

func init() {
	config := &validator.Config{TagName: "binding"}
	Validator = validator.New(config)
}

/*Connect to a database*/
func Connect(dbURL string) {
	var err error
	fmt.Println("Attempting connection to: " + dbURL)

	DB, err = gorm.Open("mysql", dbURL)
	utils.PanicOnError(err)

	DB.AutoMigrate(&Account{})
}

/*Close a database connection*/
func Close() {
	DB.Close()
}
