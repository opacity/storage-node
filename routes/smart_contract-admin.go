package routes

import (
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

func AdminSmartContractListHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminSmartContractList)
}

func AdminSmartContractEditHandler() gin.HandlerFunc {
	return ginHandlerFunc(adminSmartContractEdit)
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

func adminSmartContractList(c *gin.Context) error {
	smartContracts := []models.SmartContract{}
	results := models.DB.Find(&smartContracts)

	if results.RowsAffected == 0 {
		return NotFoundResponse(c, errors.New("no smart contracts founds"))
	}

	c.HTML(http.StatusOK, "smart-contract-list.tmpl", gin.H{
		"title":          "Smart contracts administration",
		"smartContracts": smartContracts,
	})

	return nil
}

func adminSmartContractEdit(c *gin.Context) error {
	scParam, err := strconv.ParseUint(c.Param("sc"), 10, 0)
	if err != nil {
		return InternalErrorResponse(c, errors.New("something went wrong"))
	}

	sc := models.SmartContract{}
	err = models.DB.Where("id = ?", scParam).First(&sc).Error
	if err != nil {
		return NotFoundResponse(c, err)
	}

	c.HTML(http.StatusOK, "smart-contract-edit.tmpl", gin.H{
		"title": sc.Network,
		"sc":    sc,
	})

	return nil
}

func adminSmartContractRemove(c *gin.Context) error {
	scParam, err := strconv.ParseUint(c.Param("sc"), 10, 0)
	if err != nil {
		return InternalErrorResponse(c, errors.New("something went wrong"))
	}

	err = models.DB.Where("id = ?", scParam).Delete(models.SmartContract{}).Error
	if err != nil {
		return InternalErrorResponse(c, err)
	}

	delete(services.EthWrappers, uint(scParam))

	return OkResponse(c, StatusRes{Status: "Smart contract " + c.Param("sc") + " was removed"})
}

func adminSmartContractRemoveConfirm(c *gin.Context) error {
	scParam, err := strconv.ParseUint(c.Param("sc"), 10, 0)
	if err != nil {
		return InternalErrorResponse(c, errors.New("something went wrong"))
	}

	sc := models.SmartContract{}
	err = models.DB.Where("id = ?", scParam).First(&sc).Error
	if err != nil {
		return NotFoundResponse(c, err)
	}

	c.HTML(http.StatusOK, "smart-contract-confirm-remove.tmpl", gin.H{
		"title": sc.Network,
		"sc":    sc,
	})

	return nil
}

func adminSmartContractUpdate(c *gin.Context) error {
	err := c.Request.ParseForm()
	if err != nil {
		return BadRequestResponse(c, errors.New("something went wrong"))
	}

	scParam, err := strconv.ParseUint(c.Request.PostFormValue("ID"), 10, 0)
	if err != nil {
		return InternalErrorResponse(c, err)
	}

	sc := models.SmartContract{}
	err = models.DB.Where("id = ?", scParam).First(&sc).Error
	if err != nil {
		return NotFoundResponse(c, err)
	}
	networkID, err := strconv.ParseUint(c.Request.PostFormValue("networkID"), 10, 0)
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	defaultGasPrice, err := strconv.ParseUint(c.Request.PostFormValue("defaultGasPrice"), 10, 0)
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	slowGasPrice, err := strconv.ParseUint(c.Request.PostFormValue("slowGasPrice"), 10, 0)
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	fastGasPrice, err := strconv.ParseUint(c.Request.PostFormValue("fastGasPrice"), 10, 0)
	if err != nil {
		return InternalErrorResponse(c, err)
	}

	sc.Network = c.Request.PostFormValue("network")
	sc.NetworkIDuint = uint(networkID)
	sc.NodeURL = c.Request.PostFormValue("nodeUrl")
	sc.ContractAddressString = c.Request.PostFormValue("contractAddress")
	sc.WalletAddressString = c.Request.PostFormValue("walletAddress")
	sc.DefaultGasPriceGwei = uint(defaultGasPrice)
	sc.SlowGasPriceGwei = uint(slowGasPrice)
	sc.FastGasPriceGwei = uint(fastGasPrice)

	if c.Request.PostFormValue("walletPrivateKey") != "" {
		sc.WalletPrivateKeyEncrypted = utils.EncryptWithGeneratedNonce(utils.Env.EncryptionKey, c.Request.PostFormValue("walletPrivateKey"))
	}

	if err := models.DB.Save(&sc).Error; err != nil {
		return BadRequestResponse(c, err)
	}

	walletPrivateKey, err := services.StringToPrivateKey(c.Request.PostFormValue("walletPrivateKey"))
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	services.EthWrappers[sc.ID] = &services.Eth{
		AddressNonceMap:                make(map[common.Address]uint64),
		MainWalletAddress:              services.StringToAddress(sc.WalletAddressString),
		MainWalletPrivateKey:           walletPrivateKey,
		DefaultGasPrice:                services.ConvertGweiToWei(big.NewInt(int64(sc.DefaultGasPriceGwei))),
		DefaultGasForPaymentCollection: new(big.Int).Mul(big.NewInt(int64(sc.DefaultGasPriceGwei)), big.NewInt(int64(services.GasLimitTokenSend))),
		SlowGasPrice:                   services.ConvertGweiToWei(big.NewInt(int64(sc.SlowGasPriceGwei))),
		FastGasPrice:                   services.ConvertGweiToWei(big.NewInt(int64(sc.FastGasPriceGwei))),

		ChainId:         big.NewInt(int64(sc.NetworkIDuint)),
		ContractAddress: services.StringToAddress(sc.ContractAddressString),
		NodeUrl:         sc.NodeURL,
	}

	return OkResponse(c, StatusRes{Status: fmt.Sprintf("Smart contract %d was updated", sc.ID)})
}

func adminSmartContractAdd(c *gin.Context) error {
	err := c.Request.ParseForm()
	if err != nil {
		return BadRequestResponse(c, errors.New("something went wrong"))
	}

	sc := models.SmartContract{}
	networkID, err := strconv.ParseUint(c.Request.PostFormValue("networkID"), 10, 0)
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	defaultGasPrice, err := strconv.ParseUint(c.Request.PostFormValue("defaultGasPrice"), 10, 0)
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	slowGasPrice, err := strconv.ParseUint(c.Request.PostFormValue("slowGasPrice"), 10, 0)
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	fastGasPrice, err := strconv.ParseUint(c.Request.PostFormValue("fastGasPrice"), 10, 0)
	if err != nil {
		return InternalErrorResponse(c, err)
	}

	sc.Network = c.Request.PostFormValue("network")
	sc.NetworkIDuint = uint(networkID)
	sc.NodeURL = c.Request.PostFormValue("nodeUrl")
	sc.ContractAddressString = c.Request.PostFormValue("contractAddress")
	sc.WalletAddressString = c.Request.PostFormValue("walletAddress")
	sc.WalletPrivateKeyEncrypted = utils.EncryptWithGeneratedNonce(utils.Env.EncryptionKey, c.Request.PostFormValue("walletPrivateKey"))
	sc.DefaultGasPriceGwei = uint(defaultGasPrice)
	sc.SlowGasPriceGwei = uint(slowGasPrice)
	sc.FastGasPriceGwei = uint(fastGasPrice)

	if err := models.DB.Save(&sc).Error; err != nil {
		return BadRequestResponse(c, err)
	}

	walletPrivateKey, err := services.StringToPrivateKey(c.Request.PostFormValue("walletPrivateKey"))
	if err != nil {
		return InternalErrorResponse(c, err)
	}
	services.EthWrappers[sc.ID] = &services.Eth{
		AddressNonceMap:                make(map[common.Address]uint64),
		MainWalletAddress:              services.StringToAddress(sc.WalletAddressString),
		MainWalletPrivateKey:           walletPrivateKey,
		DefaultGasPrice:                services.ConvertGweiToWei(big.NewInt(int64(sc.DefaultGasPriceGwei))),
		DefaultGasForPaymentCollection: new(big.Int).Mul(big.NewInt(int64(sc.DefaultGasPriceGwei)), big.NewInt(int64(services.GasLimitTokenSend))),
		SlowGasPrice:                   services.ConvertGweiToWei(big.NewInt(int64(sc.SlowGasPriceGwei))),
		FastGasPrice:                   services.ConvertGweiToWei(big.NewInt(int64(sc.FastGasPriceGwei))),

		ChainId:         big.NewInt(int64(sc.NetworkIDuint)),
		ContractAddress: services.StringToAddress(sc.ContractAddressString),
		NodeUrl:         sc.NodeURL,
	}

	return OkResponse(c, StatusRes{Status: fmt.Sprintf("Smart contract %d was add", sc.ID)})
}
