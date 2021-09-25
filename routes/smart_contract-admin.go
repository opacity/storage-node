package routes

import "github.com/gin-gonic/gin"

func AdminSmartContractGetHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminSmartContractGet)
}

func AdminSmartContractRemoveHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminSmartContractRemove)
}

func AdminSmartContractRemoveConfirmHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminSmartContractRemoveConfirm)
}

func AdminSmartContractUpdateHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminSmartContractUpdate)
}

func AdminSmartContractAddHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminSmartContractAdd)
}

func adminSmartContractGet(c *gin.Context) error {
	return nil
}

func adminSmartContractRemove(c *gin.Context) error {
	return nil
}

func adminSmartContractRemoveConfirm(c *gin.Context) error {
	return nil
}

func adminSmartContractUpdate(c *gin.Context) error {
	return nil
}

func adminSmartContractAdd(c *gin.Context) error {
	return nil
}
