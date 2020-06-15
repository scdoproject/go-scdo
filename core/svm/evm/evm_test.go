/**
* @file
* @copyright defined in scdo/LICENSE
 */

package evm

import (
	"math/big"

	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/common/hexutil"
	"github.com/scdoproject/go-scdo/core/state"
	"github.com/scdoproject/go-scdo/core/store"
	"github.com/scdoproject/go-scdo/crypto"
	"github.com/scdoproject/go-scdo/database/leveldb"
)

//////////////////////////////////////////////////////////////////////////////////////////////////
// PLEASE USE REMIX (OR OTHER TOOLS) TO GENERATE CONTRACT CODE AND INPUT MESSAGE.
// Online: https://remix.ethereum.org/
// Github: https://github.com/ethereum/remix-ide
//////////////////////////////////////////////////////////////////////////////////////////////////

func mustHexToBytes(hex string) []byte {
	code, err := hexutil.HexToBytes(hex)
	if err != nil {
		panic(err)
	}

	return code
}

// preprocessContract creates the contract tx dependent state DB, blockchain store
// and a default account with specified balance and nonce.
func preprocessContract(balance, nonce uint64) (*state.Statedb, store.BlockchainStore, common.Address, func()) {
	db, dispose := leveldb.NewTestDatabase()

	statedb, err := state.NewStatedb(common.EmptyHash, db)
	if err != nil {
		dispose()
		panic(err)
	}

	// Create a default account to test contract.
	addr := *crypto.MustGenerateRandomAddress()
	statedb.CreateAccount(addr)
	statedb.SetBalance(addr, new(big.Int).SetUint64(balance))
	statedb.SetNonce(addr, nonce)

	return statedb, store.NewBlockchainDatabase(db), addr, func() {
		dispose()
	}
}
