/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package core

import (
	"math/big"

	"github.com/seeledevteam/slc/common"
	"github.com/seeledevteam/slc/consensus/pow"
	"github.com/seeledevteam/slc/core/store"
	"github.com/seeledevteam/slc/core/types"
	"github.com/seeledevteam/slc/database/leveldb"
)

func newTestGenesis() *Genesis {
	accounts := map[common.Address]*big.Int{
		types.TestGenesisAccount.Addr: types.TestGenesisAccount.Amount,
	}

	return GetGenesis(NewGenesisInfo(accounts, 1, 0, big.NewInt(0), types.PowConsensus, nil))
}

func NewTestBlockchain() *Blockchain {
	return NewTestBlockchainWithVerifier(nil)
}

func NewTestBlockchainWithVerifier(verifier types.DebtVerifier) *Blockchain {
	db, _ := leveldb.NewTestDatabase()

	bcStore := store.NewCachedStore(store.NewBlockchainDatabase(db))

	genesis := newTestGenesis()
	if err := genesis.InitializeAndValidate(bcStore, db); err != nil {
		panic(err)
	}

	bc, err := NewBlockchain(bcStore, db, "", pow.NewEngine(1), verifier, -1)
	if err != nil {
		panic(err)
	}

	return bc
}
