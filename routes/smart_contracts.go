package routes

import (
	"errors"

	"github.com/gin-gonic/gin"
)

// SmartContractsHandler godoc
// @Summary gets the smart contracts address
// @Description gets the smart contracts address and related info
// @Accept json
// @Produce json
// @Success 200 {object} routes.StatusRes
// @Failure 404 {string} string "no account with that id: (with your accountID)"
// @Router /api/v2//smart-contracts [GET]
/*SmartContractsHandler is a handler for requests updating the account api version to v2*/
func SmartContractsHandler() gin.HandlerFunc {
	return ginHandlerFunc(getSmartContractsWithContext)
}

func getSmartContractsWithContext(c *gin.Context) error {
	return NotFoundResponse(c, errors.New("no smart contracts found"))
}
