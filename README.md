# storage-node

## Getting Started

The broker node uses Docker to spin up a go app, [mysql, required download](https://dev.mysql.com/downloads/file/?id=479845), and private iota instance (TODO). You must first install [Docker](https://www.docker.com/community-edition).

You should also install gopackage first and make sure you have $GOPATH set. And then clone this repo into $GOPATH/src/github.com/opacity/storage-node/<all of git repo>. This is important since we are using govendor and it only works within $GOPATH/src.

```bash
# To setup this first time, you need to have .env file. By default, use .env.test for unit test.
# Feel free to modify the .env file. Note: we don't check in .env file.
cp .env.test .env

# Starts the brokernode on port 3000
DEBUG=1 docker-compose up --build -d # This takes a few minutes when you first run it.

# You only need to pass in --build the first time, or when you make a change to the container
# This uses cached images, so it's much faster to start.
DEBUG=1 docker-compose up -d

# Note, don't include `DEBUG=1` if you would like to run a production build.
# This will have less logs and no hot reloading.
docker-compose up --build -d
docker-compose up -d
```

---

# Docker command

```bash
docker container ls # list all the running container
docker ps # list all the running container too.
docker kill storage-node_app_1 # to kill current running instance
docker logs storage-node_app_1 # print out the app's log message
docker inspect --format='{{.LogPath}}' brokernode_app_1 # print out the log's location from the docker.
sudo systemctl restart docker # restart docker
```

App in docker may running this IP address locally: http://0.0.0.0:3000/


---

# Reference library
GORM: For querying database. See https://github.com/jinzhu/gorm

Gin-Gonic: For HTTP server. See https://github.com/gin-gonic

Govendor: For Package management. See https://github.com/kardianos/govendor
