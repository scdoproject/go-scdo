/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package core

import (
	"math/big"

	"github.com/seelecredo/go-seelecredo/common"
	"github.com/seelecredo/go-seelecredo/consensus/pow"
	"github.com/seelecredo/go-seelecredo/core/store"
	"github.com/seelecredo/go-seelecredo/core/types"
	"github.com/seelecredo/go-seelecredo/database/leveldb"
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
