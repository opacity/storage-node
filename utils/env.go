package utils

import (
	"errors"
	"log"

	"os"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
)

/*StorageNodeEnv represents what our storage node environment should look like*/
type StorageNodeEnv struct {
	ProdDatabaseURL string `env:"PROD_DATABASE_URL" envDefault:""`
	TestDatabaseURL string `env:"TEST_DATABASE_URL" envDefault:""`
	DatabaseURL     string `envDefault:""`
	EncryptionKey   string `env:"ENCRYPTION_KEY" envDefault:""`
	GoEnv           string `env:"GO_ENV" envDefault:"GO_ENV not set!"`
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
}

/*SetDevelopment sets the development environment*/
func SetDevelopment() {
	initEnv()
	Env.GoEnv = "development"
	// TODO: should we have a separate development database?
	Env.DatabaseURL = Env.ProdDatabaseURL
}

/*SetTesting sets the testing environment*/
func SetTesting(filenames ...string) {
	initEnv(filenames...)
	Env.GoEnv = "test"
	Env.DatabaseURL = Env.TestDatabaseURL
}

/*IsTestEnv returns whether we are in the test environment*/
func IsTestEnv() bool {
	return Env.GoEnv == "test"
}

func tryLookUp() error {
	// TODO: this is hacky, we should clean this up
	prodDBUrl, exists := os.LookupEnv("PROD_DATABASE_URL")
	if !exists {
		return errors.New("failed to load .env variable PROD_DATABASE_URL in tryLookUp")
	}
	testDBUrl, exists := os.LookupEnv("TEST_DATABASE_URL")
	if !exists {
		return errors.New("failed to load .env variable TEST_DATABASE_URL in tryLookUp")
	}
	encryptionKey, exists := os.LookupEnv("ENCRYPTION_KEY")
	if !exists {
		return errors.New("failed to load .env variable ENCRYPTION_KEY in tryLookUp")
	}

	serverEnv := StorageNodeEnv{
		ProdDatabaseURL: prodDBUrl,
		TestDatabaseURL: testDBUrl,
		EncryptionKey:   encryptionKey,
	}

	Env = serverEnv
	return nil
}
