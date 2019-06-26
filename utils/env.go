package utils

import (
	"errors"
	"log"

	"os"

	"strconv"

	"encoding/json"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
)

const defaultAccountRetentionDays = 7

// TODO research appropriate "costInUSD" prices and include them here
// current values subject to change
const defaultPlansJson = `
{"128": {"name":"Basic","cost":2,"costInUSD":39.99,"storageInGB":128,"maxFolders":2000,"maxMetadataSizeInMB":200},
"1024": {"name":"Professional","cost":16,"costInUSD":99.99,"storageInGB":1024,"maxFolders":16000,"maxMetadataSizeInMB":1600}}
`

type PlanInfo struct {
	Name                string  `json:"name" binding:"required"`
	Cost                float64 `json:"cost" binding:"required,gt=0"`
	CostInUSD           float64 `json:"costInUSD" binding:"required,gt=0"`
	StorageInGB         int     `json:"storageInGB" binding:"required,gt=0"`
	MaxFolders          int     `json:"maxFolders" binding:"required,gt=0"`
	MaxMetadataSizeInMB int64   `json:"maxMetadataSizeInMB" binding:"required,gt=0"`
}

type PlanResponseType map[int]PlanInfo

/*StorageNodeEnv represents what our storage node environment should look like*/
type StorageNodeEnv struct {
	// Database information
	ProdDatabaseURL string `env:"PROD_DATABASE_URL" envDefault:""`
	TestDatabaseURL string `env:"TEST_DATABASE_URL" envDefault:""`
	DatabaseURL     string `envDefault:""`

	// Key to encrypt the eth private keys that we store in the accounts table
	EncryptionKey string `env:"ENCRYPTION_KEY" envDefault:""`

	// Go environment
	GoEnv string `env:"GO_ENV" envDefault:"GO_ENV not set!"`

	// Payment stuff
	ContractAddress      string `env:"TOKEN_CONTRACT_ADDRESS" envDefault:""`
	EthNodeURL           string `env:"ETH_NODE_URL" envDefault:""`
	MainWalletAddress    string `env:"MAIN_WALLET_ADDRESS" envDefault:""`
	MainWalletPrivateKey string `env:"MAIN_WALLET_PRIVATE_KEY" envDefault:""`

	// Whether the jobs should run
	EnableJobs bool `env:"ENABLE_JOB" envDefault:"false"`

	// Aws configuration
	BucketName         string `env:"AWS_BUCKET_NAME" envDefault:""`
	AwsRegion          string `env:"AWS_REGION" envDefault:""`
	AwsAccessKeyID     string `env:"AWS_ACCESS_KEY_ID" envDefault:""`
	AwsSecretAccessKey string `env:"AWS_SECRET_ACCESS_KEY" envDefault:""`

	// How long the user has to pay for their account before we delete it
	AccountRetentionDays int `env:"ACCOUNT_RETENTION_DAYS" envDefault:"7"`

	// Basic auth creds
	AdminUser     string `env:"ADMIN_USER" envDefault:""`
	AdminPassword string `env:"ADMIN_PASSWORD" envDefault:""`

	// Debug purpose
	DisplayName   string `env:"DISPLAY_NAME" envDefault:"storage-node-test"`
	SlackDebugUrl string `env:"SLACK_DEBUG_URL" envDefault:""`
	DisableDbConn bool   `env:"DISABLE_DB_CONN" envDefault:"false"`

	PlansJson string `env:"PLANS_JSON"`
	Plans     PlanResponseType

	// Stripe Keys
	StripeKeyTest string `env:"STRIPE_KEY_TEST" envDefault:"Unknown"`
	StripeKeyProd string `env:"STRIPE_KEY_PROD" envDefault:"Unknown"`
	StripeKey     string `envDefault:"Unknown"`
}

/*Env is the environment for a particular node while the application is running*/
var Env StorageNodeEnv

func initEnv(filenames ...string) {
	// Load ENV variables
	err := godotenv.Load(filenames...)
	if err != nil {
		lookupErr := tryLookUp()
		if lookupErr != nil {
			log.Fatal("Error loading environment variables: " + CollectErrors([]error{err, lookupErr}).Error())
		}
		return
	}

	storageNodeEnv := StorageNodeEnv{}
	env.Parse(&storageNodeEnv)

	if storageNodeEnv.PlansJson == "" {
		storageNodeEnv.PlansJson = defaultPlansJson
	}

	if storageNodeEnv.EncryptionKey == "" {
		log.Fatal("must set an encryption key in the .env file")
	}

	Env = storageNodeEnv
}

/*SetProduction sets the production environment*/
func SetProduction() {
	initEnv()
	Env.GoEnv = "production"
	Env.DatabaseURL = Env.ProdDatabaseURL
	Env.StripeKey = Env.StripeKeyProd
	runInitializations()
}

/*SetDevelopment sets the development environment*/
func SetDevelopment() {
	initEnv()
	Env.GoEnv = "development"
	// TODO: should we have a separate development database?
	Env.DatabaseURL = Env.ProdDatabaseURL
	Env.StripeKey = Env.StripeKeyProd
	runInitializations()
}

/*SetTesting sets the testing environment*/
func SetTesting(filenames ...string) {
	initEnv(filenames...)
	Env.PlansJson = defaultPlansJson
	Env.GoEnv = "test"
	Env.DatabaseURL = Env.TestDatabaseURL
	Env.StripeKey = Env.StripeKeyTest
	runInitializations()
}

func runInitializations() {
	InitKvStore()
	newS3Session()

	Env.Plans = make(PlanResponseType)
	err := json.Unmarshal([]byte(Env.PlansJson), &Env.Plans)
	LogIfError(err, nil)
	createPlanMetrics()
}

/*IsTestEnv returns whether we are in the test environment*/
func IsTestEnv() bool {
	return Env.GoEnv == "test"
}

/*FreeModeEnabled returns whether we are running in free mode.  Not storing this as
part of the Env because we want it to re-check each time this method is called.*/
func FreeModeEnabled() bool {
	return os.Getenv("FREE_MODE") == "true" && !IsTestEnv()
}

/*WritesEnabled returns true unless the WRITES_DISABLED env variable is set to true.  Not
storing as part of the Env because we want to re-check each time this method is called.  Not calling
this method on every endpoint because we're only trying to reject new uploads, new accounts, etc.  We
aren't trying to interrupt existing uploads or metadata sets. */
func WritesEnabled() bool {
	return IsTestEnv() || !(os.Getenv("WRITES_DISABLED") == "true")
}

func tryLookUp() error {
	var collectedErrors []error
	prodDBUrl := AppendLookupErrors("PROD_DATABASE_URL", &collectedErrors)
	testDBUrl := AppendLookupErrors("TEST_DATABASE_URL", &collectedErrors)
	encryptionKey := AppendLookupErrors("ENCRYPTION_KEY", &collectedErrors)
	contractAddress := AppendLookupErrors("TOKEN_CONTRACT_ADDRESS", &collectedErrors)
	ethNodeURL := AppendLookupErrors("ETH_NODE_URL", &collectedErrors)
	mainWalletAddress := AppendLookupErrors("MAIN_WALLET_ADDRESS", &collectedErrors)
	mainWalletPrivateKey := AppendLookupErrors("MAIN_WALLET_PRIVATE_KEY", &collectedErrors)
	bucketName := AppendLookupErrors("AWS_BUCKET_NAME", &collectedErrors)
	awsRegion := AppendLookupErrors("AWS_REGION", &collectedErrors)
	awsAccessKeyID := AppendLookupErrors("AWS_ACCESS_KEY_ID", &collectedErrors)
	awsSecretAccessKey := AppendLookupErrors("AWS_SECRET_ACCESS_KEY", &collectedErrors)
	adminUser := AppendLookupErrors("ADMIN_USER", &collectedErrors)
	adminPassword := AppendLookupErrors("ADMIN_PASSWORD", &collectedErrors)
	stripeKeyTest := AppendLookupErrors("STRIPE_KEY_TEST", &collectedErrors)
	stripeKeyProd := AppendLookupErrors("STRIPE_KEY_PROD", &collectedErrors)

	accountRetentionDaysStr := AppendLookupErrors("ACCOUNT_RETENTION_DAYS", &collectedErrors)
	accountRetentionDays, err := strconv.Atoi(accountRetentionDaysStr)
	AppendIfError(err, &collectedErrors)

	if accountRetentionDays <= 0 {
		accountRetentionDays = defaultAccountRetentionDays
	}

	plansJson, exists := os.LookupEnv("PLANS_JSON")
	if exists == false {
		plansJson = defaultPlansJson
	}

	serverEnv := StorageNodeEnv{
		ProdDatabaseURL:      prodDBUrl,
		TestDatabaseURL:      testDBUrl,
		EncryptionKey:        encryptionKey,
		ContractAddress:      contractAddress,
		EthNodeURL:           ethNodeURL,
		MainWalletAddress:    mainWalletAddress,
		MainWalletPrivateKey: mainWalletPrivateKey,
		AccountRetentionDays: accountRetentionDays,
		AwsRegion:            awsRegion,
		BucketName:           bucketName,
		AwsAccessKeyID:       awsAccessKeyID,
		AwsSecretAccessKey:   awsSecretAccessKey,
		AdminUser:            adminUser,
		AdminPassword:        adminPassword,
		PlansJson:            plansJson,
		StripeKeyTest:        stripeKeyTest,
		StripeKeyProd:        stripeKeyProd,
	}

	Env = serverEnv
	return CollectErrors(collectedErrors)
}

/*AppendLookupErrors will look up an environment variable, and if there was an
error, it will append it to the array of errors that are passed in*/
func AppendLookupErrors(property string, collectedErrors *[]error) string {
	value, exists := os.LookupEnv(property)
	if !exists || value == "" {
		AppendIfError(errors.New("in tryLookup, failed to load .env variable: "+property), collectedErrors)
	}
	return value
}
