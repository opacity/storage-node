package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
)

type SmartContractResp struct {
	Contracts []SmartContractRespEntity `json:"contracts"`
}

type SmartContractRespEntity struct {
	Network string `json:"network"`
	Address string `json:"address"`
	ChainID uint   `json:"chainId"`
}

// SmartContractsHandler godoc
// @Summary gets the smart contracts address
// @Description gets the smart contracts address and related info
// @Accept json
// @Produce json
// @Success 200 {array} SmartContractResp
// @Failure 404 {string} string "no account with that id: (with your accountID)"
// @Router /api/v2/smart-contracts [GET]
/*SmartContractsHandler is a handler for requests updating the account api version to v2*/
func SmartContractsHandler() gin.HandlerFunc {
	return ginHandlerFunc(getSmartContractsWithContext)
}

func getSmartContractsWithContext(c *gin.Context) error {
	smartContracts, err := models.GetAllSmartContracts()
	scResp := []SmartContractRespEntity{}

	if err != nil {
		return NotFoundResponse(c, err)
	}

	for _, sc := range smartContracts {
		scResp = append(scResp, SmartContractRespEntity{
			Network: sc.Network,
			Address: sc.ContractAddressString,
			ChainID: sc.NetworkIDuint,
		})
	}

	return OkResponse(c, SmartContractResp{
		Contracts: scResp,
	})
}
