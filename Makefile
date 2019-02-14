# Reference makefile:  https://github.com/moorereason/hugo/blob/master/Makefile

PACKAGE = github.com/opacity/storage-node

govendor:
	go get -u github.com/kardianos/govendor
	govendor sync

# this is hacky, we should do something better
ethereum-fix:
	rm -rf ./vendor/github.com/ethereum/go-ethereum/crypto/secp256k1/
	govendor fetch github.com/ethereum/go-ethereum/crypto/secp256k1/^