# Reference makefile:  https://github.com/moorereason/hugo/blob/master/Makefile

PACKAGE = github.com/opacity/storage-node

govendor:
	go get -u github.com/kardianos/govendor
	govendor sync ${PACKAGE}