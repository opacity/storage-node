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

/*StorageNodeEnv represents what our storage node environment should look like*/
type StorageNodeEnv struct {
	ProdDatabaseURL      string `env:"PROD_DATABASE_URL" envDefault:""`
	TestDatabaseURL      string `env:"TEST_DATABASE_URL" envDefault:""`
	DatabaseURL          string `envDefault:""`
	EncryptionKey        string `env:"ENCRYPTION_KEY" envDefault:""`
	GoEnv                string `env:"GO_ENV" envDefault:"GO_ENV not set!"`
	ContractAddress      string `env:"TOKEN_CONTRACT_ADDRESS" envDefault:""`
	EthNodeURL           string `env:"ETH_NODE_URL" envDefault:""`
	MainWalletAddress    string `env:"MAIN_WALLET_ADDRESS" envDefault:""`
	MainWalletPrivateKey string `env:"MAIN_WALLET_PRIVATE_KEY" envDefault:""`
	DisplayName          string `env:"DISPLAY_NAME" envDefault:"storage-node-test"`
	EnableJobs           bool   `env:"ENABLE_JOB" envDefault:"false"`

	// Aws configuration
	BucketName         string `env:"AWS_BUCKET_NAME" envDefault:""`
	AwsRegion          string `env:"AWS_REGION" envDefault:""`
	AwsAccessKeyID     string `env:"AWS_ACCESS_KEY_ID" envDefault:""`
	AwsSecretAccessKey string `env:"AWS_SECRET_ACCESS_KEY" envDefault:""`

	// Debug purpose
	SlackDebugUrl string `env:"SLACK_DEBUG_URL" envDefault:""`
	DisableDbConn bool   `env:"DISABLE_DB_CONN" envDefault:"false"`

	AccountRetentionDays int `env:"ACCOUNT_RETENTION_DAYS" envDefault:"7"`
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

/*SetProduction sets the production environment*/
func SetProduction() {
	initEnv()
	Env.GoEnv = "production"
	Env.DatabaseURL = Env.ProdDatabaseURL
	InitKvStore()
	newS3Session()
}

/*SetDevelopment sets the development environment*/
func SetDevelopment() {
	initEnv()
	Env.GoEnv = "development"
	// TODO: should we have a separate development database?
	Env.DatabaseURL = Env.ProdDatabaseURL
	InitKvStore()
	newS3Session()
}

/*SetTesting sets the testing environment*/
func SetTesting(filenames ...string) {
	initEnv(filenames...)
	Env.GoEnv = "test"
	Env.DatabaseURL = Env.TestDatabaseURL
	InitKvStore()
	newS3Session()
}

/*IsTestEnv returns whether we are in the test environment*/
func IsTestEnv() bool {
	return Env.GoEnv == "test"
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
	accountRetentionDaysStr := AppendLookupErrors("ACCOUNT_RETENTION_DAYS", &collectedErrors)
	accountRetentionDays, err := strconv.Atoi(accountRetentionDaysStr)
	if err != nil {
		collectedErrors = append(collectedErrors, err)
	}
	if accountRetentionDays <= 0 {
		accountRetentionDays = defaultAccountRetentionDays
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
	}

	Env = serverEnv
	return CollectErrors(collectedErrors)
}

/*AppendLookupErrors will look up an environment variable, and if there was an
error, it will append it to the array of errors that are passed in*/
func AppendLookupErrors(property string, collectedErrors *[]error) string {
	value, exists := os.LookupEnv(property)
	if !exists {
		*collectedErrors = append(*(collectedErrors),
			errors.New("in tryLookup, failed to load .env variable: "+property))
	}
	return value
}
