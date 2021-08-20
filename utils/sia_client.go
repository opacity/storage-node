package utils

import (
	"errors"
	"strings"

	sia_modules "gitlab.com/NebulousLabs/Sia/modules"
	sia_api "gitlab.com/NebulousLabs/Sia/node/api"
	sia_client "gitlab.com/NebulousLabs/Sia/node/api/client"
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
	siaClient = sia_client.New(opts)
}

func IsSiaClientInit() error {
	if siaClient == nil {
		return errors.New("sia client is not initialized.")
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
