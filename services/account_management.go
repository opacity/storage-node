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
	CheckIfPending
}

/*CheckIfPaid defines the required parameters and return values for
an instance of AccountManagement's CheckIfPaid method*/
type CheckIfPaid func(common.Address, *big.Int, uint) (bool, error)

/*CheckIfPending defines the required parameters and return values for
an instance of AccountManagement's CheckIfPending method*/
type CheckIfPending func(common.Address, uint) bool

/*BackendManagement is an instance of AccountManagement which we will use
for backend-managed subscriptions*/
var BackendManagement AccountManagement

func init() {
	BackendManagement = AccountManagement{
		CheckIfPaid:    checkIfBackendSubscriptionPaid,
		CheckIfPending: checkIfBackendSubscriptionPaymentPending,
	}
}

func checkIfBackendSubscriptionPaid(address common.Address, amount *big.Int, networkID uint) (bool, error) {
	var tokenBalance *big.Int
	if tokenBalance = EthWrappers[networkID].GetTokenBalance(address); tokenBalance == big.NewInt(-1) {
		return false, errors.New("could not get balance")
	}

	return tokenBalance.Cmp(amount) >= 0, nil
}

func checkIfBackendSubscriptionPaymentPending(address common.Address, networkID uint) bool {
	return EthWrappers[networkID].CheckForPendingTokenTxs(address)
}
