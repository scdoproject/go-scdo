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

	/* Scdo will fork from ScdoForkHeight,
	   Below is the seele block information before forkHeight
	*/

	if info.ShardNumber == 1 {
		previousBlockHash = common.StringToHash("0x4384e84ecc14d26d5ba35aa8c2fe1cf0b952f8ea512690131656640be99759fb")
		// creator, _ = common.HexToAddress("0xdf5b4c0f52a8ec4b697c35046ba7fb9b26416891")
		createTimestamp = big.NewInt(1594607812)
		txHash = common.StringToHash("0x147a99f2f80d8d13a0eddb80d1fa3aaf072be5108e1f78a59358857977de6a7d")
		// stateHash = common.StringToHash("0xeb83242b992027f86a14c05bbfe3b704c605b83b994d295ac51eae218bd15f26")
	}
	if info.ShardNumber == 2 {
		previousBlockHash = common.StringToHash("0x5d8b650caccc314704d895d5485fa6d5be242f7282584f2dda868a5fb0bc8858")
		// creator, _ = common.HexToAddress("0xa71b2b2fde959f33edb9a6940b3dc0c6771820b1")
		createTimestamp = big.NewInt(1594748542)
		txHash = common.StringToHash("0xdf847512609aa19753397d70c21c99672016130f113b94f44a9a63d3f45cff12")
		// stateHash = common.StringToHash("0xb2ef42c3898ca872232c1757f6e9c827bfbe32de959a362b61e63789d313f05e")
	}
	if info.ShardNumber == 3 {
		previousBlockHash = common.StringToHash("0xb72ddc18bc087da97381710b6f1a3f52a783b5d5cdd47d652ee1fd8e4cb3d152")
		// creator, _ = common.HexToAddress("0x8aeeeec186d64db4712921b06c4ecfeac7476461")
		createTimestamp = big.NewInt(1594027090)
		txHash = common.StringToHash("0xdf0e38040963e5e6dffdc5bc9aa65f1c3511f230c7425bc45527337aadc18dd3")
		// stateHash = common.StringToHash("0x832a90891732b63008237168b1fe6188d0d611f0d3044e12aa5b7ea0169bf3b4")
	}
	if info.ShardNumber == 4 {
		previousBlockHash = common.StringToHash("0xc96df35f12f04c42a3050a6e5336690dcb8311b9ac432ae79f9a8dd8ca68b1a9")
		// creator, _ = common.HexToAddress("0xef3e6426f207fc27182ab07d22f40ba61cde9cc1")
		createTimestamp = big.NewInt(1594230091)
		txHash = common.StringToHash("0x762967d93f059c69752e7f97dcd56e44d2eefd49f6ee8db1ca2f9f4417358626")
		// stateHash = common.StringToHash("0x5d745d82bc1dd5b1b2067da3112174f8527b66ce8d0a92748bdf3730b99a54e9")
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
		info.Masteraccount, _ = common.HexToAddress("1S01f1bb5c799305bcf3e7c1316445757a517ab291")
		info.Balance = minedRewardsPerShard
		statedb.CreateAccount(info.Masteraccount)
		statedb.SetBalance(info.Masteraccount, info.Balance)
	} else if info.ShardNumber == 2 {
		info.Masteraccount, _ = common.HexToAddress("2S02fb048755bd1f35d035406a6aab3c771f6e51c1")
		info.Balance = minedRewardsPerShard
		statedb.CreateAccount(info.Masteraccount)
		statedb.SetBalance(info.Masteraccount, info.Balance)
	} else if info.ShardNumber == 3 {
		info.Masteraccount, _ = common.HexToAddress("3S03a43b0c0c524e9a2f98bd605615e49d58c96491")
		info.Balance = minedRewardsPerShard
		statedb.CreateAccount(info.Masteraccount)
		statedb.SetBalance(info.Masteraccount, info.Balance)
	} else if info.ShardNumber == 4 {
		info.Masteraccount, _ = common.HexToAddress("4S04e58416cf2973ad208a797a2c115292d0166d01")
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
