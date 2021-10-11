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
	if DB != nil {
		DB.Close()
	}
	var err error
	fmt.Println("Attempting connection to: " + dbURL)

	DB, err = gorm.Open("mysql", dbURL)
	utils.PanicOnError(err)

	// List all the schema
	DB.AutoMigrate(&Account{})
	DB.AutoMigrate(&File{})
	DB.AutoMigrate(&SiaProgressFile{})
	DB.AutoMigrate(&S3ObjectLifeCycle{})
	DB.AutoMigrate(&CompletedFile{})
	DB.AutoMigrate(&CompletedUploadIndex{})
	DB.AutoMigrate(&StripePayment{})
	DB.AutoMigrate(&Upgrade{})
	DB.AutoMigrate(&Renewal{})
	DB.AutoMigrate(&ExpiredAccount{})
	DB.AutoMigrate(&PublicShare{})
	DB.AutoMigrate(&SmartContract{})
	// @TODO: uncomment after planID migration
	// DB.AutoMigrate(&utils.PlanInfo{})
}

// Temporary func @TODO: remove after migration
func MigratePlanIds() error {
	initPlans := []utils.PlanInfo{}
	DB.Model(&utils.PlanInfo{}).Find(&initPlans)
	DB.DropTable(utils.PlanInfo{})

	DB.AutoMigrate(&utils.PlanInfo{})

	DB.Model(&Account{}).AddForeignKey("plan_info_id", "plan_infos(id)", "RESTRICT", "CASCADE")

	for planId, plan := range initPlans {
		plan.ID = uint(planId) + 1
		plan.MonthsInSubscription = 12
		DB.Model(&utils.PlanInfo{}).Create(&plan)

		// migrate Accounts to PlanInfo
		MigrateAccountsToPlanId(plan.ID, plan.StorageInGB)
		MigrateUpgradeToPlanIdNew(plan.ID, plan.StorageInGB)
		MigrateUpgradeToPlanIdOld(plan.ID, plan.StorageInGB)
	}

	DB.Model(&Upgrade{}).AddForeignKey("new_plan_info_id", "plan_infos(id)", "RESTRICT", "CASCADE")
	DB.Exec("ALTER TABLE upgrades DROP PRIMARY KEY, ADD PRIMARY KEY (`account_id`, `new_plan_info_id`)")

	// drop 'storage_location' and 'storage_limit'
	DB.Model(&Account{}).DropColumn("storage_location")
	DB.Model(&Account{}).DropColumn("storage_limit")

	DB.Model(&Upgrade{}).DropColumn("new_storage_limit")
	DB.Model(&Upgrade{}).DropColumn("old_storage_limit")
	DB.Model(&Upgrade{}).DropColumn("duration_in_months")
	DB.Model(&Upgrade{}).DropColumn("opq_cost")

	DB.Model(&Renewal{}).DropColumn("opq_cost")

	return utils.SetPlansMigration(true)
}

/*Close a database connection*/
func Close() {
	DB.Close()
}
