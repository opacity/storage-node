package utils

import (
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

func SiaFileMetadataByPath(fileSiaPath string) (sia_api.RenterFile, error) {
	siaPath, err := sia_modules.NewSiaPath(fileSiaPath)
	if err != nil {
		return sia_api.RenterFile{}, err
	}

	return siaClient.RenterFileGet(siaPath)
}
