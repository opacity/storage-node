package main

import (
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/routes"
	"github.com/opacity/storage-node/utils"
)

func main() {
	defer models.Close()

	//utils.SetProduction()
	utils.SetDevelopment()

	models.Connect(utils.Env.DatabaseURL)

	routes.CreateRoutes()
}
