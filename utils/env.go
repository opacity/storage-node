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
const TestFileStoragePerMetadataInMB = 100
const TestMaxPerMetadataSizeInMB = 50

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

	// Folder metadata restrictions

	// FileStoragePerMetadataInMB is how many MBs worth of files they have to have stored
	// to create one additional folder metadata, i.e. if the value is 100, they need to have 101 MB
	// of files stored before they're allowed 2 metadatas.

	// MaxPerMetadataSizeInMB is the maximum size of each metadata.  We will use this
	// in the accounts model to derive the max metadata storage for each plan size.
	FileStoragePerMetadataInMB int `env:"FILE_STORAGE_PER_METADATA_IN_MB" envDefault:"100"`
	MaxPerMetadataSizeInMB     int `env:"MAX_PER_METADATA_SIZE_IN_MB" envDefault:"50"`
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
	Env.FileStoragePerMetadataInMB = TestFileStoragePerMetadataInMB
	Env.MaxPerMetadataSizeInMB = TestMaxPerMetadataSizeInMB
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
	adminUser := AppendLookupErrors("ADMIN_USER", &collectedErrors)
	adminPassword := AppendLookupErrors("ADMIN_PASSWORD", &collectedErrors)

	accountRetentionDaysStr := AppendLookupErrors("ACCOUNT_RETENTION_DAYS", &collectedErrors)
	accountRetentionDays, err := strconv.Atoi(accountRetentionDaysStr)
	AppendIfError(err, &collectedErrors)

	fileStoragePerMetadataInMBStr := AppendLookupErrors("FILE_STORAGE_PER_METADATA_IN_MB", &collectedErrors)
	fileStoragePerMetadataInMB, err := strconv.Atoi(fileStoragePerMetadataInMBStr)
	AppendIfError(err, &collectedErrors)

	maxPerMetadataSizeInMBStr := AppendLookupErrors("MAX_PER_METADATA_SIZE_IN_MB", &collectedErrors)
	maxPerMetadataSizeInMB, err := strconv.Atoi(maxPerMetadataSizeInMBStr)
	AppendIfError(err, &collectedErrors)

	if accountRetentionDays <= 0 {
		accountRetentionDays = defaultAccountRetentionDays
	}

	serverEnv := StorageNodeEnv{
		ProdDatabaseURL:            prodDBUrl,
		TestDatabaseURL:            testDBUrl,
		EncryptionKey:              encryptionKey,
		ContractAddress:            contractAddress,
		EthNodeURL:                 ethNodeURL,
		MainWalletAddress:          mainWalletAddress,
		MainWalletPrivateKey:       mainWalletPrivateKey,
		AccountRetentionDays:       accountRetentionDays,
		AwsRegion:                  awsRegion,
		BucketName:                 bucketName,
		AwsAccessKeyID:             awsAccessKeyID,
		AwsSecretAccessKey:         awsSecretAccessKey,
		AdminUser:                  adminUser,
		AdminPassword:              adminPassword,
		FileStoragePerMetadataInMB: fileStoragePerMetadataInMB,
		MaxPerMetadataSizeInMB:     maxPerMetadataSizeInMB,
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
