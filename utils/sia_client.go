package utils

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	sia_modules "gitlab.com/NebulousLabs/Sia/modules"
	sia_api "gitlab.com/NebulousLabs/Sia/node/api"
	sia_client "gitlab.com/NebulousLabs/Sia/node/api/client"
	sia_types "gitlab.com/NebulousLabs/Sia/types"
)

const (
	dataPieces   = 0
	parityPieces = 0
)

var siaClient *sia_client.Client

func InitSiaClient() {
	opts, err := sia_client.DefaultOptions()
	if err != nil {
		LogIfError(err, nil)
	}

	opts.Password = Env.SiaApiPassword
	opts.Address = Env.SiaNodeAddress

	siaClient = sia_client.New(opts)
}

func IsSiaClientInit() error {
	if siaClient == nil {
		return errors.New("sia client is not initialized")
	}

	return nil
}

func UploadSiaFile(fileData, fileSiaPath string, deleteIfExisting bool) error {
	siaPath, err := sia_modules.NewSiaPath(fileSiaPath)
	if err != nil {
		return err
	}

	fileDataReader := strings.NewReader(fileData)

	err = siaClient.RenterUploadStreamPost(fileDataReader, siaPath, dataPieces, parityPieces, deleteIfExisting)
	if err != nil {
		return err
	}

	return nil
}

func GetSiaFileMetadataByPath(fileSiaPath string) (sia_api.RenterFile, error) {
	siaPath, err := sia_modules.NewSiaPath(fileSiaPath)
	if err != nil {
		return sia_api.RenterFile{}, err
	}

	return siaClient.RenterFileGet(siaPath)
}

func DeleteSiaFile(fileSiaPath string) error {
	siaPath, err := sia_modules.NewSiaPath(fileSiaPath)
	if err != nil {
		return err
	}

	return siaClient.RenterFileDeletePost(siaPath)
}

func GetSiaAddress() string {
	return siaClient.Address
}

func GetSiaRenter() sia_api.RenterGET {
	rg, err := siaClient.RenterGet()
	if err != nil {
		LogIfError(err, nil)
	}
	return rg
}

func SetSiaAllowance(funds, periodWeeks, hosts, renewWindowWeeks, expectedStorageTB, expectedDownloadTB, expectedUploadTB string) error {

	hastings, err := sia_types.ParseCurrency(funds + "SC")
	if err != nil {
		return err
	}
	var fundsCurrency sia_types.Currency
	_, err = fmt.Sscan(hastings, &fundsCurrency)
	if err != nil {
		return err
	}

	periodInt, err := strconv.Atoi(periodWeeks)
	if err != nil {
		return err
	}
	period := periodInt * int(sia_types.BlocksPerWeek)

	hostsInt, err := strconv.Atoi(hosts)
	if err != nil {
		return err
	}

	renewWindowInt, err := strconv.Atoi(renewWindowWeeks)
	if err != nil {
		return err
	}
	renewWindow := renewWindowInt * int(sia_types.BlocksPerWeek)

	expectedStorage, err := parseTB(expectedStorageTB)
	if err != nil {
		return err
	}

	expectedUpload, err := parseTB(expectedUploadTB)
	if err != nil {
		return err
	}
	expectedUploadPerMonth := expectedUpload / uint64(sia_types.BlocksPerMonth)

	expectedDownload, err := parseTB(expectedDownloadTB)
	if err != nil {
		return err
	}
	expectedDownloadPerMonth := expectedDownload / uint64(sia_types.BlocksPerMonth)

	allowanceReq := siaClient.RenterPostPartialAllowance()
	allowanceReq = allowanceReq.WithFunds(fundsCurrency)
	allowanceReq = allowanceReq.WithHosts(uint64(hostsInt))
	allowanceReq = allowanceReq.WithPeriod(sia_types.BlockHeight(period))
	allowanceReq = allowanceReq.WithRenewWindow(sia_types.BlockHeight(renewWindow))
	allowanceReq = allowanceReq.WithExpectedStorage(expectedStorage)
	allowanceReq = allowanceReq.WithExpectedUpload(expectedUploadPerMonth)
	allowanceReq = allowanceReq.WithExpectedDownload(expectedDownloadPerMonth)

	// return allowanceReq.Send()
	return nil
}

func parseTB(value string) (uint64, error) {
	valueRat, ok := new(big.Rat).SetString(value)
	if !ok {
		return 0, errors.New("malformed value")
	}
	u := valueRat.Mul(valueRat, new(big.Rat).SetInt(big.NewInt(1e12))).RatString()
	var uintValue uint64
	_, err := fmt.Sscan(u, &uintValue)
	if err != nil {
		return 0, err
	}

	return uintValue, nil
}

func GetWalletInfo() sia_api.WalletGET {
	wallet, err := siaClient.WalletGet()
	if err != nil {
		LogIfError(err, nil)
	}
	return wallet
}

func IsSiaSynced() bool {
	cg, err := siaClient.ConsensusGet()
	if err != nil {
		LogIfError(err, nil)
	}

	return cg.Synced
}
