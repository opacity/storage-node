package utils

import (
	"errors"
	"log"

	"os"

	"strconv"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
)

const defaultAccountRetentionDays = 7
const defaultStripeRetentionDays = 30

const DefaultPlansJson = `{
	"10":{"name":"Free","cost":0,"costInUSD":0.00,"storageInGB":10,"maxFolders":200,"maxMetadataSizeInMB":20},
	"128":{"name":"Basic","cost":2,"costInUSD":39.99,"storageInGB":128,"maxFolders":2000,"maxMetadataSizeInMB":200},
	"1024":{"name":"Professional","cost":16,"costInUSD":79.99,"storageInGB":1024,"maxFolders":16000,"maxMetadataSizeInMB":1600},
	"2048":{"name":"Business","cost":32,"costInUSD":9.99,"storageInGB":2048,"maxFolders":32000,"maxMetadataSizeInMB":3200},
	"10000":{"name":"Custom10TB","cost":150000,"costInUSD":550.00,"storageInGB":10000,"maxFolders":156000,"maxMetadataSizeInMB":15600}
}`

type PlanInfo struct {
	Name                string  `gorm:"primary_key" json:"name"`
	Cost                float64 `json:"cost"`
	CostInUSD           float64 `json:"costInUSD"`
	StorageInGB         int     `json:"storageInGB"`
	MaxFolders          int     `json:"maxFolders"`
	MaxMetadataSizeInMB int64   `json:"maxMetadataSizeInMB"`
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

	// How long to retain a stripe payment before we delete it
	StripeRetentionDays int `env:"STRIPE_RETENTION_DAYS" envDefault:"30"`

	// Basic auth creds
	AdminUser     string `env:"ADMIN_USER" envDefault:""`
	AdminPassword string `env:"ADMIN_PASSWORD" envDefault:""`

	// Debug purpose
	DisplayName   string `env:"DISPLAY_NAME" envDefault:"storage-node-test"`
	SlackDebugUrl string `env:"SLACK_DEBUG_URL" envDefault:""`
	DisableDbConn bool   `env:"DISABLE_DB_CONN" envDefault:"false"`

	Plans PlanResponseType

	// Stripe Keys
	StripeKeyTest string `env:"STRIPE_KEY_TEST" envDefault:"Unknown"`
	StripeKeyProd string `env:"STRIPE_KEY_PROD" envDefault:"Unknown"`
	StripeKey     string `envDefault:"Unknown"`

	// Whether accepting credit cards is enabled
	EnableCreditCards bool `env:"ENABLE_CREDIT_CARDS" envDefault:"false"`
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

	if storageNodeEnv.EncryptionKey == "" {
		log.Fatal("must set an encryption key in the .env file")
	}

	Env = storageNodeEnv
}

/*SetLive sets the live environment*/
func SetLive() {
	initEnv()
	Env.GoEnv = "live"
	Env.DatabaseURL = Env.ProdDatabaseURL
	Env.StripeKey = Env.StripeKeyProd
	runInitializations()
}

/*SetTesting sets the testing environment*/
func SetTesting(filenames ...string) {
	initEnv(filenames...)
	Env.GoEnv = "test"
	Env.DatabaseURL = Env.TestDatabaseURL
	Env.StripeKey = Env.StripeKeyTest
	runInitializations()
}

func runInitializations() {
	InitKvStore()
	newS3Session()
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

	stripeRetentionDaysStr := AppendLookupErrors("STRIPE_RETENTION_DAYS", &collectedErrors)
	stripeRetentionDays, err := strconv.Atoi(stripeRetentionDaysStr)
	AppendIfError(err, &collectedErrors)

	if stripeRetentionDays <= 0 {
		stripeRetentionDays = defaultStripeRetentionDays
	}

	enableCreditCardsStr, _ := os.LookupEnv("ENABLE_CREDIT_CARDS")
	enableCreditCards := enableCreditCardsStr == "true"

	serverEnv := StorageNodeEnv{
		ProdDatabaseURL:      prodDBUrl,
		TestDatabaseURL:      testDBUrl,
		EncryptionKey:        encryptionKey,
		ContractAddress:      contractAddress,
		EthNodeURL:           ethNodeURL,
		MainWalletAddress:    mainWalletAddress,
		MainWalletPrivateKey: mainWalletPrivateKey,
		AccountRetentionDays: accountRetentionDays,
		StripeRetentionDays:  stripeRetentionDays,
		AwsRegion:            awsRegion,
		BucketName:           bucketName,
		AwsAccessKeyID:       awsAccessKeyID,
		AwsSecretAccessKey:   awsSecretAccessKey,
		AdminUser:            adminUser,
		AdminPassword:        adminPassword,
		StripeKeyTest:        stripeKeyTest,
		StripeKeyProd:        stripeKeyProd,
		EnableCreditCards:    enableCreditCards,
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
