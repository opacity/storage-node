package models

import (
	_ "database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/utils"
	"gopkg.in/go-playground/validator.v8"
)

var (
	DB        *gorm.DB
	Validator *validator.Validate
)

func init() {
	config := &validator.Config{TagName: "binding"}
	Validator = validator.New(config)
}

/*Connect to a database*/
func Connect(dbUrl string) {
	var err error
	fmt.Println("Attempting connection to: " + dbUrl)

	DB, err = gorm.Open("mysql", dbUrl)
	utils.PanicOnError(err)
}

/*Close a database connection*/
func Close() {
	DB.Close()
}
