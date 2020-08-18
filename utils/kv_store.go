package utils

import (
	"errors"
	"io/ioutil"
	"time"

	"fmt"

	"os"

	"github.com/dgraph-io/badger"
)

const badgerDirProd = "/var/lib/badger/prod"

/*TestValueTimeToLive is some default value we can use in unit
tests for K:V pairs in badger*/
const TestValueTimeToLive = 1 * time.Minute

// Singleton DB
var badgerDB *badger.DB
var dbNoInitError error
var badgerDirTest string

/*KVPairs is a type.  Map key strings to value strings*/
type KVPairs map[string]string

/*KVKeys is a type.  An array of key strings*/
type KVKeys []string

func init() {
	dbNoInitError = errors.New("badgerDB not initialized, Call InitKvStore() first")

	badgerDirTest, _ = ioutil.TempDir("", "badgerForUnitTest")
}

/*InitKvStore returns db so that caller can call CloseKvStore to close it when it is done.*/
func InitKvStore() (err error) {
	if badgerDB != nil {
		return nil
	}

	// Setup opts
	var opts badger.Options

	if IsTestEnv() {
		opts = badger.DefaultOptions(badgerDirTest).WithTruncate(true)
	} else {
		opts = badger.DefaultOptions(badgerDirProd).WithTruncate(true)
	}

	badgerDB, err = badger.Open(opts)
	LogIfError(err, nil)
	return err
}

/*CloseKvStore closes the db.*/
func CloseKvStore() error {
	if badgerDB == nil {
		return nil
	}

	err := badgerDB.Close()
	LogIfError(err, nil)
	badgerDB = nil
	return err
}

/*RemoveAllKvStoreData removes all the data. Caller should call InitKvStore() again to create a new one.*/
func RemoveAllKvStoreData() error {
	if err := CloseKvStore(); err != nil {
		return err
	}

	var dir string
	if IsTestEnv() {
		dir = badgerDirTest
	} else {
		dir = badgerDirProd
	}
	err := os.RemoveAll(dir)
	LogIfError(err, map[string]interface{}{"badgerDir": dir})
	return err
}

/*GetValueFromKV gets a single value from the provided key*/
func GetValueFromKV(key string) (value string, expirationTime time.Time, err error) {
	expirationTime = time.Now()
	if badgerDB == nil {
		return value, expirationTime, dbNoInitError
	}

	err = badgerDB.View(func(txn *badger.Txn) error {
		if key == "" {
			return errors.New("no key specified")
		}

		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		value = ""
		if item != nil {
			var valBytes []byte
			err := item.Value(func(val []byte) error {
				if val == nil {
					valBytes = nil
				} else {
					valBytes = append([]byte{}, val...)
				}
				return nil
			})
			if err != nil {
				return err
			}

			value = string(valBytes)
			expirationTime = time.Unix(int64(item.ExpiresAt()), 0)
		}

		return nil
	})
	if err != badger.ErrKeyNotFound {
		LogIfError(err, nil)
	}

	return
}

/*BatchGet returns KVPairs for a set of keys. It won't treat Key missing as error.*/
func BatchGet(ks *KVKeys) (kvs *KVPairs, err error) {
	kvs = &KVPairs{}
	if badgerDB == nil {
		return kvs, dbNoInitError
	}

	err = badgerDB.View(func(txn *badger.Txn) error {
		for _, k := range *ks {
			// Skip any empty keys.
			if k == "" {
				continue
			}

			item, err := txn.Get([]byte(k))
			if err == badger.ErrKeyNotFound {
				continue
			}
			if err != nil {
				return err
			}

			fmt.Println(item.UserMeta())

			val := ""
			if item != nil {
				var valBytes []byte
				err := item.Value(func(val []byte) error {
					if val == nil {
						valBytes = nil
					} else {
						valBytes = append([]byte{}, val...)
					}
					return nil
				})
				if err != nil {
					return err
				}

				val = string(valBytes)
			}

			// Mutate KV map
			(*kvs)[k] = val
		}

		return nil
	})
	LogIfError(err, map[string]interface{}{"batchSize": len(*ks)})

	return
}

/*BatchSet updates a set of KVPairs. Return error if any fails.*/
func BatchSet(kvs *KVPairs, ttl time.Duration) error {
	ttl = getTTL(ttl)
	if badgerDB == nil {
		return dbNoInitError
	}

	var err error
	txn := badgerDB.NewTransaction(true)
	for k, v := range *kvs {
		if k == "" {
			err = errors.New("BatchSet does not accept key as empty string")
			break
		}

		e := txn.SetEntry(badger.NewEntry([]byte(k), []byte(v)).WithTTL(ttl).WithDiscard())
		if e == nil {
			continue
		}

		if e == badger.ErrTxnTooBig {
			e = nil
			if commitErr := txn.Commit(); commitErr != nil {
				e = commitErr
			} else {
				txn = badgerDB.NewTransaction(true)
				e = txn.SetEntry(badger.NewEntry([]byte(k), []byte(v)).WithTTL(ttl))
			}
		}

		if e != nil {
			err = e
			break
		}
	}

	defer txn.Discard()
	if err == nil {
		err = txn.Commit()
	}
	LogIfError(err, map[string]interface{}{"batchSize": len(*kvs)})
	return err
}

/*BatchDelete deletes a set of KVKeys, Return error if any fails.*/
func BatchDelete(ks *KVKeys) error {
	if badgerDB == nil {
		return dbNoInitError
	}

	var err error
	txn := badgerDB.NewTransaction(true)
	for _, key := range *ks {
		e := txn.Delete([]byte(key))
		if e == nil {
			continue
		}

		if e == badger.ErrTxnTooBig {
			e = nil
			if commitErr := txn.Commit(); commitErr != nil {
				e = commitErr
			} else {
				txn = badgerDB.NewTransaction(true)
				e = txn.Delete([]byte(key))
			}
		}

		if e != nil {
			err = e
			break
		}
	}

	defer txn.Discard()
	if err == nil {
		err = txn.Commit()
	}

	LogIfError(err, map[string]interface{}{"batchSize": len(*ks)})
	return err
}

func Iterate() error {
	err := badgerDB.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.AllVersions = true
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()

			expirationTime := time.Unix(int64(item.ExpiresAt()), 0)
			fmt.Println(expirationTime.Format(time.RFC822))

			timeStringToParse := "17 Oct 21 15:04 MST"
			newExpirationTime, _ := time.Parse(time.RFC822, timeStringToParse)

			//if !item.IsDeletedOrExpired() {
			//	continue
			//}

			//if !expirationTime.Before(time.Now()) {
			//	continue
			//}

			if !expirationTime.Before(newExpirationTime) {
				fmt.Println("skip")
				continue
			}

			if string(k) == "" {
				continue
			}

			var valCopy []byte
			err := item.Value(func(val []byte) error {
				fmt.Printf("key=%s\n", k)
				valCopy = append([]byte{}, val...)
				return nil
			})

			if err != nil {
				return err
			}

			err = BatchSet(&KVPairs{string(k): string(valCopy)}, time.Until(newExpirationTime))

			if err != nil {
				fmt.Println(err)
				return err
			}
		}
		return nil
	})
	return err
}

func getTTL(ttl time.Duration) time.Duration {
	if !IsTestEnv() {
		return ttl
	}
	return TestValueTimeToLive
}
