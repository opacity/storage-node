# storage-node

## Getting Started

The storage node uses Docker to spin up a go app, [mysql, required download](https://dev.mysql.com/downloads/file/?id=479845), and private iota instance (TODO). You must first install [Docker](https://www.docker.com/community-edition).

You should also install gopackage first and make sure you have $GOPATH set. And then clone this repo into $GOPATH/src/github.com/opacity/storage-node/<all of git repo>. This is important since we are using govendor and it only works within $GOPATH/src.

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
govendor sync # make sure to include any dependence
docker-compose exec app govendor test +local

# Manage new dependence
govendor list # will list all of new dependence. "m"=missing
govendor fetch github.com/...  # fetch the dependence and add it to your local vendor folder
# Govendor is not smart enough to fetch all sub dependence, so you might end to manually fetching all its sub-dependence yourself.
govendor test +local # test to build it
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

Govendor: For Package management. See https://github.com/kardianos/govendor


# ENV
When you pull down changes for the app, check for new properties that have been added to
env.go to add them to your environment file.  Also note that the aws s3 libraries will check
your .env file for env variables such as AWS_BUCKET_NAME, AWS_ACCESS_KEY_ID, AWS_BUCKET_REGION,
and AWS_SECRET_ACCESS_KEY.  So these .env files must be present to do s3 uploads even if it
is not immediately obvious from looking at the storage node code.


