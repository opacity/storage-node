package services

import (
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
type CheckIfPaid func(common.Address, *big.Int) (bool, uint, error)

/*CheckIfPending defines the required parameters and return values for
an instance of AccountManagement's CheckIfPending method*/
type CheckIfPending func(common.Address) bool

/*BackendManagement is an instance of AccountManagement which we will use
for backend-managed subscriptions*/
var BackendManagement AccountManagement

func init() {
	BackendManagement = AccountManagement{
		CheckIfPaid:    checkIfBackendSubscriptionPaid,
		CheckIfPending: checkIfBackendSubscriptionPaymentPending,
	}
}

func checkIfBackendSubscriptionPaid(address common.Address, amount *big.Int) (bool, uint, error) {
	for networkID := range EthWrappers {
		tokenBalance := EthOpsWrapper.GetTokenBalance(EthWrappers[networkID], address)
		if tokenBalance.Cmp(amount) >= 0 {
			return true, networkID, nil
		}
	}

	return false, 0, nil
}

func checkIfBackendSubscriptionPaymentPending(address common.Address) bool {
	for networkID := range EthWrappers {
		if EthOpsWrapper.CheckForPendingTokenTxs(EthWrappers[networkID], address) {
			return true
		}
	}

	return false
}
