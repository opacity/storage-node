# storage-node

## Getting Started

The storage node uses Docker to spin up a go app [mysql, required download](https://dev.mysql.com/downloads/file/?id=479845).
You must first install [Docker](https://www.docker.com/community-edition).

You should also install gopackage first and make sure you have `$GOPATH` set. The go version should be 1.11 or greater.
Then clone this repo somewhere outside of the `$GOPATH`.  It needs to be outside of the `$GOPATH` so that the go commands
will use go modules.

```bash
# To setup this first time, you need to have .env file. By default, use .env.test for unit test.
# Feel free to modify the .env file. Note: we don't check in .env file.
cp .env.test .env

# Starts the storage node on port 3000
DEBUG=1 docker-compose up --build -d # This takes a few minutes when you first run it.

# You only need to pass in --build the first time, or when you make a change to the container
# This uses cached images, so it's much faster to start.
DEBUG=1 docker-compose up -d

# Note, don't include `DEBUG=1` if you would like to run a production build.
# This will have less logs and no hot reloading.
docker-compose up --build -d
docker-compose up -d

# Run unit test
docker-compose exec app go test ./...

# Manage new dependencies
# If you have installed the project outside of the $GOPATH and your go version is high enough, 
commands such as:
go build 
# and: 
go test 
# ...should automatically add new dependencies 
# as needed and update your go.mod file.
# To install specific versions you can use commands such as:  
go get foo@v1.2.3
go get foo@master
go get foo@e3702bed2
# Or, directly edit the go.mod file.
```

---

# Docker command

```bash
docker container ls # list all the running container
docker ps # list all the running container too.
docker kill storage-node_app_1 # to kill current running instance
docker logs storage-node_app_1 # print out the app's log message
docker inspect --format='{{.LogPath}}' storage-node_app_1 # print out the log's location from the docker.
sudo systemctl restart docker # restart docker
```

App in docker may running this IP address locally: http://0.0.0.0:3000/

App metrics can be found in http://0.0.0.0:3000/admin/metrics

---

# Useful testing

Use postman to do request. Send request as POST, 0.0.0.0:3000/api/v1/accounts with JSON as body:
{"accountID":"abc", "storageLimit":8, "durationInMonths": 2}

---

# Reference library
GORM: For querying database. See https://github.com/jinzhu/gorm

Gin-Gonic: For HTTP server. See https://github.com/gin-gonic

Go Modules: For dependency management. See:  
https://github.com/golang/go/wiki/Modules
https://www.kablamo.com.au/blog/2018/12/10/just-tell-me-how-to-use-go-modules
https://ukiahsmith.com/blog/a-gentle-introduction-to-golang-modules/
https://arslan.io/2018/08/26/using-go-modules-with-vendor-support-on-travis-ci/
https://dave.cheney.net/2018/07/16/using-go-modules-with-travis-ci

# ENV
When you pull down changes for the app, check for new properties that have been added to
env.go to add them to your environment file.  Also note that the aws s3 libraries will check
your .env file for env variables such as AWS_BUCKET_NAME, AWS_ACCESS_KEY_ID, AWS_BUCKET_REGION,
and AWS_SECRET_ACCESS_KEY.  So these .env files must be present to do s3 uploads even if it
is not immediately obvious from looking at the storage node code.


