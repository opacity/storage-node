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

func SetEnvFromParamStore(go_env string) {
	newParamStoreSession()

	params, nextToken := getParametersByPath("", go_env)
	setEnvParamsFromStore(params)

	for nextToken != nil {
		params, nextToken = getParametersByPath(*nextToken, go_env)
		setEnvParamsFromStore(params)
	}
}

func getParametersByPath(nextToken, go_env string) ([]*ssm.Parameter, *string) {
	parametersBypathInput := &ssm.GetParametersByPathInput{
		Path:           aws.String("/storage-node/" + go_env + "/"),
		Recursive:      aws.Bool(true),
		WithDecryption: aws.Bool(true),
	}

	if nextToken != "" {
		parametersBypathInput.SetNextToken(nextToken)
	}
	paramResp, err := paramStoreSvc.paramStore.GetParametersByPath(parametersBypathInput)

	if err != nil {
		PanicOnError(err)
	}

	return paramResp.Parameters, paramResp.NextToken
}

func setEnvParamsFromStore(params []*ssm.Parameter) {
	for _, param := range params {
		paramFQDName := strings.Split(*param.Name, "/")
		os.Setenv(paramFQDName[len(paramFQDName)-1], *param.Value)
	}
}
