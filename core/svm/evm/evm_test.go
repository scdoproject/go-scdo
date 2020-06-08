/**
* @file
* @copyright defined in slc/LICENSE
 */

package evm

import (
	"math/big"

	"github.com/seeledevteam/slc/common"
	"github.com/seeledevteam/slc/common/hexutil"
	"github.com/seeledevteam/slc/core/state"
	"github.com/seeledevteam/slc/core/store"
	"github.com/seeledevteam/slc/crypto"
	"github.com/seeledevteam/slc/database/leveldb"
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
