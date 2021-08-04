package utils

import (
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

type paramStoreWrapper struct {
	paramStore *ssm.SSM
}

var paramStoreSvc *paramStoreWrapper

func newParamStoreSession() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	paramStoreSvc = &paramStoreWrapper{
		paramStore: ssm.New(sess),
	}
}

func LoadAllParams(go_env string) {
	newParamStoreSession()

	params, err := paramStoreSvc.paramStore.GetParametersByPath(&ssm.GetParametersByPathInput{
		Path:           aws.String("/" + go_env + "/"),
		Recursive:      aws.Bool(true),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		PanicOnError(err)
	}

	for _, param := range params.Parameters {
		paramFQDName := strings.Split(*param.Name, "/")
		os.Setenv(*param.Name, paramFQDName[len(paramFQDName)-1])
	}
}
