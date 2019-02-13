package models

import (
	_ "database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/utils"
)

var (
	/*DB is our database connection*/
	DB *gorm.DB
)

func init() {
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
