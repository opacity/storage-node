package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/getsentry/sentry-go"
	"github.com/opacity/storage-node/jobs"
	"github.com/opacity/storage-node/models"
	"github.com/opacity/storage-node/routes"
	"github.com/opacity/storage-node/services"
	"github.com/opacity/storage-node/utils"
)

var GO_ENV = "localhost"
var VERSION = "local"

func main() {
	if GO_ENV == "" {
		utils.PanicOnError(errors.New("the GO_ENV variable is not set; application can not run"))
	}
	os.Setenv("GO_ENV", GO_ENV)
	if GO_ENV == "production" || GO_ENV == "dev2" {
		tracesSampleRate := 0.3
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              "https://03e807e8312d47938a94b73ebec3cc84@o126495.ingest.sentry.io/5855671",
			Release:          VERSION,
			Environment:      GO_ENV,
			AttachStacktrace: true,
			TracesSampleRate: tracesSampleRate,
			BeforeSend:       sentryOpacityBeforeSend,
		})
		if err != nil {
			log.Fatalf("sentry.Init: %s", err)
		}
		defer sentry.Flush(5 * time.Second)
	}

	defer catchError()
	defer models.Close()

	utils.SetLive()

	err := services.InitStripe(utils.Env.StripeKeyProd)
	utils.PanicOnError(err)

	utils.SlackLog("Begin to restart service!")

	if !utils.Env.DisableDbConn {
		models.Connect(utils.Env.DatabaseURL)
	}

	setEnvPlans()
	migrateEnvWallets()

	jobs.StartupJobs()
	if utils.Env.EnableJobs {
		jobs.ScheduleBackgroundJobs()
	}

	routes.CreateRoutes()
}

func SetWallets() {
	smartContracts := []models.SmartContract{}
	models.DB.Find(&smartContracts)

	defaultGasPrice := services.ConvertGweiToWei(big.NewInt(80))

	services.EthOpsWrapper = services.EthOps{
		TransferToken:           services.TransferTokenWrapper,
		TransferETH:             services.TransferETHWrapper,
		GetTokenBalance:         services.GetTokenBalanceWrapper,
		GetETHBalance:           services.GetETHBalanceWrapper,
		CheckForPendingTokenTxs: services.CheckForPendingTokenTxsWrapper,
	}

	for _, smartContract := range smartContracts {
		// singletons
		services.EthWrappers[smartContract.ID] = &services.Eth{
			AddressNonceMap:                make(map[common.Address]uint64),
			DefaultGasPrice:                services.ConvertGweiToWei(big.NewInt(80)),
			DefaultGasForPaymentCollection: new(big.Int).Mul(defaultGasPrice, big.NewInt(int64(services.GasLimitTokenSend))),
			SlowGasPrice:                   services.ConvertGweiToWei(big.NewInt(80)),
			FastGasPrice:                   services.ConvertGweiToWei(big.NewInt(145)),

			ChainId:         smartContract.NetworkID,
			ContractAddress: smartContract.ContractAddress,
			NodeUrl:         smartContract.NodeURL,
		}
	}
}

func setEnvPlans() {
	plans := []utils.PlanInfo{}
	results := models.DB.Find(&plans)

	utils.Env.Plans = make(utils.PlanResponseType)

	if results.RowsAffected == 0 {
		err := json.Unmarshal([]byte(utils.DefaultPlansJson), &utils.Env.Plans)
		utils.LogIfError(err, nil)

		for _, plan := range utils.Env.Plans {
			models.DB.Model(&utils.PlanInfo{}).Create(&plan)
		}
	} else {
		for _, plan := range plans {
			utils.Env.Plans[plan.StorageInGB] = plan
		}
	}

	utils.CreatePlanMetrics()
}

// @TODO: remove this after first run with wallets in DB
func migrateEnvWallets() {
	wallets := []models.SmartContract{}
	walletsResults := models.DB.Find(&wallets)

	if walletsResults.RowsAffected == 0 {
		ethMainWallet := models.SmartContract{
			Network:                   "ethereum",
			NetworkIDuint:             1,
			ContractAddressString:     utils.Env.ContractAddress,
			NodeURL:                   utils.Env.EthNodeURL,
			WalletAddressString:       utils.Env.MainWalletAddress,
			WalletPrivateKeyEncrypted: utils.EncryptWithoutNonce(utils.Env.EncryptionKey, utils.Env.MainWalletPrivateKey),
			DefaultGasPriceGwei:       80,
			SlowGasPriceGwei:          80,
			FastGasPriceGwei:          145,
		}

		polygonMainWallet := models.SmartContract{
			Network:                   "polygon",
			NetworkIDuint:             137,
			ContractAddressString:     utils.Env.PolygonContractAddress,
			NodeURL:                   utils.Env.PolygonNodeURL,
			WalletAddressString:       utils.Env.PolygonMainWalletAddress,
			WalletPrivateKeyEncrypted: utils.EncryptWithoutNonce(utils.Env.EncryptionKey, utils.Env.PolygonMainWalletPrivateKey),
			DefaultGasPriceGwei:       80,
			SlowGasPriceGwei:          80,
			FastGasPriceGwei:          145,
		}

		if GO_ENV != "production" {
			ethMainWallet.Network = "goerli"
			ethMainWallet.NetworkIDuint = 5

			polygonMainWallet.Network = "mumbai"
			polygonMainWallet.NetworkIDuint = 80001
		}
		models.DB.Model(&utils.PlanInfo{}).Create(&ethMainWallet)
		models.DB.Model(&utils.PlanInfo{}).Create(&polygonMainWallet)
	}
	SetWallets()
}

func sentryOpacityBeforeSend(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	if event.Request != nil {
		req := routes.GenericRequest{}

		if err := json.Unmarshal([]byte(event.Request.Data), &req); err == nil {
			if len(event.Exception) > 0 {
				frames := event.Exception[0].Stacktrace.Frames
				// do not include http/gin-gonic and the Sentry throw funcs ones
				event.Exception[0].Stacktrace.Frames = frames[6 : len(frames)-3]
			}

			event.Request.Data = req.RequestBody
		}
	}

	return event
}

func catchError() {
	// Capture the error
	if r := recover(); r != nil {
		sentry.CurrentHub().Recover(r)

		buff := bytes.NewBufferString("")
		buff.Write(debug.Stack())
		stacks := strings.Split(buff.String(), "\n")

		threadId := stacks[0]
		if len(stacks) > 5 {
			stacks = stacks[5:] // skip the Stack() and Defer method.
		}
		utils.SlackLogError(fmt.Sprintf("Crash due to err %v!!!\nRunning on thread: %s,\nStack: \n%v\n", r, threadId, strings.Join(stacks, "\n")))
	}
}
