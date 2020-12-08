/**
* @file
* @copyright defined in scdo/LICENSE
 */

package evm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params"
	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/core/store"
	"github.com/scdoproject/go-scdo/core/types"
	"github.com/scdoproject/go-scdo/core/vm"
)

// NewEVMByDefaultConfig returns a new EVM. The returned EVM is not thread safe and should
// only ever be used *once*.
func NewEVMByDefaultConfig(tx *types.Transaction, statedb *StateDB, blockHeader *types.BlockHeader, bcStore store.BlockchainStore) *vm.EVM {
	evmContext := newEVMContext(tx, blockHeader, blockHeader.Creator, bcStore)
	chainConfig := &params.ChainConfig{
		ChainID:             big.NewInt(1),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(int64(common.EmeryForkHeight)),
		IstanbulBlock:       big.NewInt(int64(common.EmeryForkHeight)),
		Ethash:              new(params.EthashConfig),
	}
	vmConfig := &vm.Config{}

	return vm.NewEVM(*evmContext, statedb, chainConfig, *vmConfig)
}

// NewEVMContext creates a new context for use in the EVM.
func newEVMContext(tx *types.Transaction, header *types.BlockHeader, minerAddress common.Address, bcStore store.BlockchainStore) *vm.Context {
	canTransferFunc := func(db vm.StateDB, addr common.Address, amount *big.Int) bool {
		return db.GetBalance(addr).Cmp(amount) >= 0
	}

	transferFunc := func(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
		db.SubBalance(sender, amount)

		if sender.Shard() == recipient.Shard() {
			db.AddBalance(recipient, amount)
		}
	}

	heightToHashMapping := map[uint64]common.Hash{
		header.Height - 1: header.PreviousBlockHash,
	}
	getHashFunc := func(height uint64) common.Hash {
		for preHash := header.PreviousBlockHash; ; {
			if hash, ok := heightToHashMapping[height]; ok {
				return hash
			}

			preHeader, err := bcStore.GetBlockHeader(preHash)
			if err != nil {
				return common.EmptyHash
			}

			heightToHashMapping[preHeader.Height-1] = preHeader.PreviousBlockHash
			preHash = preHeader.PreviousBlockHash
		}
	}

	return &vm.Context{
		CanTransfer: canTransferFunc,
		Transfer:    transferFunc,
		GetHash:     getHashFunc,
		Origin:      tx.Data.From,
		Coinbase:    minerAddress,
		BlockNumber: new(big.Int).SetUint64(header.Height),
		Time:        new(big.Int).Set(header.CreateTimestamp),
		Difficulty:  new(big.Int).Set(header.Difficulty),
	}
}
