package models

import (
	/*blank import to make drivers available*/
	_ "database/sql"
	"fmt"
	/*blank import to make drivers available*/
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

var (
	/*DB is our connection to the database*/
	DB *gorm.DB

	/*BackendManager is a copy of services.BackendManagement.  We can
	stub out methods in unit tests*/
	BackendManager = services.BackendManagement
)

/*Connect to a database*/
func Connect(dbURL string) {
	var err error
	fmt.Println("Attempting connection to: " + dbURL)

	DB, err = gorm.Open("mysql", dbURL)
	utils.PanicOnError(err)

	schema := []interface{}{
		Account{},
		File{},
		S3ObjectLifeCycle{},
	}
	for _, s := range schema {
		DB.AutoMigrate(&s)
	}
}

/*Close a database connection*/
func Close() {
	DB.Close()
}
