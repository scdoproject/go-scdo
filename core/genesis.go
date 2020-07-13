/**
* @file
* @copyright defined in scdo/LICENSE
 */

package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/common/errors"
	"github.com/scdoproject/go-scdo/consensus"
	"github.com/scdoproject/go-scdo/core/state"
	"github.com/scdoproject/go-scdo/core/store"
	"github.com/scdoproject/go-scdo/core/types"
	"github.com/scdoproject/go-scdo/crypto"
	"github.com/scdoproject/go-scdo/database"
	leveldbErrors "github.com/syndtr/goleveldb/leveldb/errors"
)

var (
	// ErrGenesisHashMismatch is returned when the genesis block hash between the store and memory mismatch.
	ErrGenesisHashMismatch = errors.New("genesis block hash mismatch")

	// ErrGenesisNotFound is returned when genesis block not found in the store.
	ErrGenesisNotFound = errors.New("genesis block not found")
)

const genesisBlockHeight = common.ScdoForkHeight

// Genesis represents the genesis block in the blockchain.
type Genesis struct {
	header *types.BlockHeader
	info   *GenesisInfo
}

// GenesisInfo genesis info for generating genesis block, it could be used for initializing account balance
type GenesisInfo struct {
	// Accounts accounts info for genesis block used for test
	// map key is account address -> value is account balance
	Accounts map[common.Address]*big.Int `json:"accounts,omitempty"`

	// Difficult initial difficult for mining. Use bigger difficult as you can. Because block is chosen by total difficult
	Difficult int64 `json:"difficult"`

	// ShardNumber is the shard number of genesis block.
	ShardNumber uint `json:"shard"`

	// CreateTimestamp is the initial time of genesis
	CreateTimestamp *big.Int `json:"timestamp"`

	// Consensus consensus type
	Consensus types.ConsensusType `json:"consensus"`

	// Validators istanbul consensus validators
	Validators []common.Address `json:"validators"`

	// master account
	Masteraccount common.Address `json:"master"`

	// balance of the master account
	Balance *big.Int `json:"balance"`
}

func NewGenesisInfo(accounts map[common.Address]*big.Int, difficult int64, shard uint, timestamp *big.Int,
	consensus types.ConsensusType, validator []common.Address) *GenesisInfo {

	var masteraccount common.Address
	var balance *big.Int
	// if shard == 1 {
	// 	masteraccount, _ = common.HexToAddress("1S01b04cb8be750904e2c1912417afbf1f3bc61a51")
	// 	balance = big.NewInt(17500000000000000)
	// } else if shard == 2 {
	// 	masteraccount, _ = common.HexToAddress("2S02b04cb8be750904e2c1912417afbf1f3bc61a51")
	// 	balance = big.NewInt(17500000000000000)
	// } else if shard == 3 {
	// 	masteraccount, _ = common.HexToAddress("3S03b04cb8be750904e2c1912417afbf1f3bc61a51")
	// 	balance = big.NewInt(17500000000000000)
	// } else if shard == 4 {
	// 	masteraccount, _ = common.HexToAddress("4S04b04cb8be750904e2c1912417afbf1f3bc61a51")
	// 	balance = big.NewInt(17500000000000000)
	// } else {
	// 	masteraccount, _ = common.HexToAddress("0S0000000000000000000000000000000000000000")
	// 	balance = big.NewInt(0)
	// }
	return &GenesisInfo{
		Accounts:        accounts,
		Difficult:       difficult,
		ShardNumber:     shard,
		CreateTimestamp: timestamp,
		Consensus:       consensus,
		Validators:      validator,
		Masteraccount:   masteraccount,
		Balance:         balance,
	}
}

// Hash returns GenesisInfo hash
func (info *GenesisInfo) Hash() common.Hash {
	data, err := json.Marshal(info)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal err: %s", err))
	}

	return crypto.HashBytes(data)
}

// shardInfo represents the extra data that saved in the genesis block in the blockchain.
type shardInfo struct {
	ShardNumber uint
}

// GetGenesis gets the genesis block according to accounts' balance
func GetGenesis(info *GenesisInfo) *Genesis {
	if info.Difficult <= 0 {
		info.Difficult = 1
	}

	statedb := getStateDB(info)
	stateRootHash, err := statedb.Hash()
	if err != nil {
		panic(err)
	}

	extraData := []byte{}
	if info.Consensus == types.IstanbulConsensus {
		extraData = generateConsensusInfo(info.Validators)
	}

	shard := common.SerializePanic(shardInfo{
		ShardNumber: info.ShardNumber,
	})

	previousBlockHash := common.EmptyHash
	creator := common.EmptyAddress
	stateHash := stateRootHash
	txHash := types.MerkleRootHash(nil)
	createTimestamp := info.CreateTimestamp
	// Scdo will fork from ScdoForkHeight,
	// Below is the seele block information before forkHeight

	if info.ShardNumber == 1 {
		previousBlockHash = common.StringToHash("0x31906ed8685385b2c0a5fd47a731bfdf323120ed3998dd9e9b09081ba2893bba")
		// creator, _ = common.HexToAddress("0xcd2da0aabfcfe5c6d8f968a6dab3dbab21650931")
		createTimestamp = big.NewInt(1594465904)
		txHash = common.StringToHash("0x138539eca2287bbef155db690fb8b1b697a27f50d83fb12aeb7c74d3b8ecd035")
		// stateHash = common.StringToHash("0x17027f48e843817fcf0ac5db0be6b6c45a2f7197b23b5593592a8cecceb36010")
	}
	if info.ShardNumber == 2 {
		previousBlockHash = common.StringToHash("0x0e87f3aa429e999181629b9f36d3f2c22e2a910d0799ff9bbcdc7705b328e23c")
		// creator, _ = common.HexToAddress("0xebdd6c53aaed41dc48f358c05027207229f56bc1")
		createTimestamp = big.NewInt(1594606936)
		txHash = common.StringToHash("0xe166fa486fe5d612e9e02d58f26ece3e036017edf1f69ff118a187bbf3436aa3")
		// stateHash = common.StringToHash("0xb30b802fd8d455035da1d981b72ea0500526d41e97abb610cac2951f3923805f")
	}
	if info.ShardNumber == 3 {
		previousBlockHash = common.StringToHash("0x28ec6a67af2bd9c744c423682850e80f2b938fbc263bd9231f5afb5477929c1c")
		// creator, _ = common.HexToAddress("0xe5c5a01d776ce738aae49f84425ae8ba0ccea2c1")
		createTimestamp = big.NewInt(1593886151)
		txHash = common.StringToHash("0xecd03c41bbdf4abf84eab125a4c2ee36de76f841dc457758aca91d3006669598")
		// stateHash = common.StringToHash("0xd2b128028cc86af129b16b68ee6fa6313805d4bc36a06d073dff2cc2b4bd459c")
	}
	if info.ShardNumber == 4 {
		previousBlockHash = common.StringToHash("0xb16f96ba74e41f01a8a89cc17617b18daa46499daea611306f06573437a8d182")
		// creator, _ = common.HexToAddress("0x3bce94f9fe99d5464d0505ea67d9ee5009c2a851")
		createTimestamp = big.NewInt(1594088340)
		txHash = common.StringToHash("0x2412151d95434f7fbd80c7c69a8076494185670f7e32022ef4304817f4c75b5c")
		// stateHash = common.StringToHash("0xcc6a4c2973b434098df477d4e129f00b2c02a6125b068a1712793f9a4457233e")
	}
	return &Genesis{
		header: &types.BlockHeader{
			PreviousBlockHash: previousBlockHash, // Note: this blockhash is seele block=2818931 hash
			Creator:           creator,
			StateHash:         stateHash,
			TxHash:            txHash,
			Difficulty:        big.NewInt(info.Difficult),
			Height:            genesisBlockHeight,
			CreateTimestamp:   createTimestamp,
			Consensus:         info.Consensus,
			Witness:           shard,
			ExtraData:         extraData,
		},
		info: info,
	}
}

func generateConsensusInfo(addrs []common.Address) []byte {
	var consensusInfo []byte
	consensusInfo = append(consensusInfo, bytes.Repeat([]byte{0x00}, types.IstanbulExtraVanity)...)

	ist := &types.IstanbulExtra{
		Validators:    addrs,
		Seal:          []byte{},
		CommittedSeal: [][]byte{},
	}

	istPayload, err := rlp.EncodeToBytes(&ist)
	if err != nil {
		panic("failed to encode istanbul extra")
	}

	consensusInfo = append(consensusInfo, istPayload...)
	return consensusInfo
}

// GetShardNumber gets the shard number of genesis
func (genesis *Genesis) GetShardNumber() uint {
	return genesis.info.ShardNumber
}

// InitializeAndValidate writes the genesis block in the blockchain store if unavailable.
// Otherwise, check if the existing genesis block is valid in the blockchain store.
func (genesis *Genesis) InitializeAndValidate(bcStore store.BlockchainStore, accountStateDB database.Database) error {
	storedGenesisHash, err := bcStore.GetBlockHash(genesisBlockHeight)

	if err == leveldbErrors.ErrNotFound {
		return genesis.store(bcStore, accountStateDB)
	}

	if err != nil {
		return errors.NewStackedErrorf(err, "failed to get block hash by height %v in canonical chain", genesisBlockHeight)
	}

	storedGenesis, err := bcStore.GetBlock(storedGenesisHash)
	if err != nil {
		return errors.NewStackedErrorf(err, "failed to get genesis block by hash %v", storedGenesisHash)
	}

	data, err := getShardInfo(storedGenesis)
	if err != nil {
		return errors.NewStackedError(err, "failed to get extra data in genesis block")
	}

	if data.ShardNumber != genesis.info.ShardNumber {
		return fmt.Errorf("specific shard number %d does not match with the shard number in genesis info %d", data.ShardNumber, genesis.info.ShardNumber)
	}

	if headerHash := genesis.header.Hash(); !headerHash.Equal(storedGenesisHash) {
		return ErrGenesisHashMismatch
	}

	return nil
}

// store atomically stores the genesis block in the blockchain store.
func (genesis *Genesis) store(bcStore store.BlockchainStore, accountStateDB database.Database) error {
	statedb := getStateDB(genesis.info)

	batch := accountStateDB.NewBatch()
	if _, err := statedb.Commit(batch); err != nil {
		return errors.NewStackedError(err, "failed to commit batch into statedb")
	}

	if err := batch.Commit(); err != nil {
		return errors.NewStackedError(err, "failed to commit batch into database")
	}

	if err := bcStore.PutBlockHeader(genesis.header.Hash(), genesis.header, genesis.header.Difficulty, true); err != nil {
		return errors.NewStackedError(err, "failed to put genesis block header into store")
	}

	return nil
}

func getStateDB(info *GenesisInfo) *state.Statedb {
	statedb := state.NewEmptyStatedb(nil)

	curReward := consensus.GetReward(common.ScdoForkHeight)
	var minedRewardsPerShard = big.NewInt(0)
	minedRewardsPerShard.Mul(curReward, big.NewInt(common.ScdoForkHeight))

	if info.ShardNumber == 1 {
		info.Masteraccount, _ = common.HexToAddress("1S01b04cb8be750904e2c1912417afbf1f3bc61a51")
		info.Balance = minedRewardsPerShard
		statedb.CreateAccount(info.Masteraccount)
		statedb.SetBalance(info.Masteraccount, info.Balance)
	} else if info.ShardNumber == 2 {
		info.Masteraccount, _ = common.HexToAddress("2S02b04cb8be750904e2c1912417afbf1f3bc61a51")
		info.Balance = minedRewardsPerShard
		statedb.CreateAccount(info.Masteraccount)
		statedb.SetBalance(info.Masteraccount, info.Balance)
	} else if info.ShardNumber == 3 {
		info.Masteraccount, _ = common.HexToAddress("3S03b04cb8be750904e2c1912417afbf1f3bc61a51")
		info.Balance = minedRewardsPerShard
		statedb.CreateAccount(info.Masteraccount)
		statedb.SetBalance(info.Masteraccount, info.Balance)
	} else if info.ShardNumber == 4 {
		info.Masteraccount, _ = common.HexToAddress("4S04b04cb8be750904e2c1912417afbf1f3bc61a51")
		info.Balance = minedRewardsPerShard
		statedb.CreateAccount(info.Masteraccount)
		statedb.SetBalance(info.Masteraccount, info.Balance)
	} else {
		info.Masteraccount, _ = common.HexToAddress("0S0000000000000000000000000000000000000000")
		info.Balance = big.NewInt(0)
	}

	for addr, amount := range info.Accounts {
		if !common.IsShardEnabled() || addr.Shard() == info.ShardNumber {
			statedb.CreateAccount(addr)
			statedb.SetBalance(addr, amount)
		}
	}

	return statedb
}

// getShardInfo returns the extra data of specified genesis block.
func getShardInfo(genesisBlock *types.Block) (*shardInfo, error) {
	if genesisBlock.Header.Height != genesisBlockHeight {
		return nil, fmt.Errorf("invalid genesis block height %v", genesisBlock.Header.Height)
	}

	data := &shardInfo{}
	if err := common.Deserialize(genesisBlock.Header.Witness, data); err != nil {
		return nil, errors.NewStackedError(err, "failed to deserialize the extra data of genesis block")
	}

	return data, nil
}
