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

	/*EthWrapper is a copy of services.EthWrapper*/
	EthWrapper = services.EthWrapper
)

/*Connect to a database*/
func Connect(dbURL string) {
	if DB != nil {
		DB.Close()
	}
	var err error
	fmt.Println("Attempting connection to: " + dbURL)

	DB, err = gorm.Open("mysql", dbURL)
	utils.PanicOnError(err)

	// List all the schema
	DB.AutoMigrate(&Account{})
	DB.AutoMigrate(&File{}).ModifyColumn("aws_object_key", "text")
	DB.AutoMigrate(&S3ObjectLifeCycle{})
	DB.AutoMigrate(&CompletedFile{})
	DB.AutoMigrate(&CompletedUploadIndex{})
	DB.AutoMigrate(&StripePayment{})
	DB.AutoMigrate(&Upgrade{})
	DB.AutoMigrate(&Renewal{})
	DB.AutoMigrate(&ExpiredAccount{})
}

/*Close a database connection*/
func Close() {
	DB.Close()
}
