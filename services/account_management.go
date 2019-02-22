package services

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

/*AccountManagement defines what an account manager's methods look like.
Eventually we'll have two different account managers.  One for backend managed
accounts and one for smart contract managed accounts*/
type AccountManagement struct {
	CheckIfPaid
}

/*CheckIfPaid defines the required parameters and return values for
an instance of AccountManagement's CheckIfPaid method*/
type CheckIfPaid func(common.Address, *big.Int) (bool, error)

/*BackendManagement is an instance of AccountManagement which we will use
for backend-managed subscriptions*/
var BackendManagement AccountManagement

func init() {
	BackendManagement = AccountManagement{
		CheckIfPaid: checkIfBackendSubscriptionPaid,
	}
}

func checkIfBackendSubscriptionPaid(address common.Address, amount *big.Int) (bool, error) {
	var tokenBalance *big.Int
	if tokenBalance = EthWrapper.GetTokenBalance(address); tokenBalance == big.NewInt(-1) {
		return false, errors.New("could not get balance")
	}

	return tokenBalance.Int64() >= amount.Int64(), nil
}
