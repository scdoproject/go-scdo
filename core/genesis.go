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
		previousBlockHash = common.StringToHash("0xc439dd3398fb4d7596cce6382d18cacf1b873a49680959e0267f7588c591cacb")
		// creator, _ = common.HexToAddress("0xde0c88a825e3d049853de9be6a188a1c1e591411")
		createTimestamp = big.NewInt(1596764398)
		// stateHash = common.StringToHash("0xa5ea0d7e9a17c5a92f4919963d63d0ce2b597c2ef5b2256c89bc72214c11c304")
		txHash = common.StringToHash("0x9a43f0cacb52cae451defd3452cdd86b70373edca6dd724ff77e3b6c93f4b97e")
	}
	if info.ShardNumber == 2 {
		previousBlockHash = common.StringToHash("0xa3f5dddb003600eb0a717fca3c234c93c21ceaac88cdb611cbce42eaa4f2645b")
		// creator, _ = common.HexToAddress("0x544053ec780cb701c319d892edd540bb94f0d4b1")
		createTimestamp = big.NewInt(1596928094)
		// stateHash = common.StringToHash("0x67221984616afef08f7e0e9036a9d0b3c2f39e1f5295e4c32779c5d4649e3c3a")
		txHash = common.StringToHash("0x8cead9e6cb9a9ca9299d4dd26208b800cb9b3d10f0ff9fab96ee90060517a199")
	}
	if info.ShardNumber == 3 {
		previousBlockHash = common.StringToHash("0xfc1b5faa1a9a64f7479184ebf541659882f4ff6b2c0539bb36aec1b428bf2299")
		// creator, _ = common.HexToAddress("0x0ead1657dec87b3af3316f1379f34661a8715711")
		createTimestamp = big.NewInt(1596174170)
		// stateHash = common.StringToHash("0x0b73ee4a1da5b60b1efa865ed707e7bc720f9c507b92d3a1eff06bb727e46c38")
		txHash = common.StringToHash("0xf9fd5e150c980a356a34ca0290965a8a2d5b8b5290c3216ba5d0974932af8ac1")
	}
	if info.ShardNumber == 4 {
		previousBlockHash = common.StringToHash("0x3e2833eb7769f7f1881c364014ab662228fa3f6a6af669d15cea4b3cab974e16")
		// creator, _ = common.HexToAddress("0x9b2413f544122c23e93b6cdce6fec0ebb981d421")
		createTimestamp = big.NewInt(1596385932)
		// stateHash = common.StringToHash("0x8fc546cf5e52ac4e5158c7b262aa0b366752913bfb12bb2eb38a3e6eb31d7439")
		txHash = common.StringToHash("0x6453d364115e975bd5824fdd84beb5c995170db5575677724b026fe7516888cc")
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
