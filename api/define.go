package api

import (
	"math/big"

	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/core/state"
	"github.com/scdoproject/go-scdo/core/store"
	"github.com/scdoproject/go-scdo/core/types"
	"github.com/scdoproject/go-scdo/log"
	"github.com/scdoproject/go-scdo/p2p"
	"github.com/scdoproject/go-scdo/rpc"
)

// Backend interface provides the common API services (that are provided by
// both full and light clients) with access to necessary functions.
type Backend interface {
	GetP2pServer() *p2p.Server
	GetNetVersion() string
	GetNetWorkID() string

	TxPoolBackend() Pool
	ChainBackend() Chain
	ProtocolBackend() Protocol
	Log() *log.ScdoLog
	IsSyncing() bool

	GetBlock(hash common.Hash, height int64) (*types.Block, error)
	GetBlockTotalDifficulty(hash common.Hash) (*big.Int, error)
	GetReceiptByTxHash(txHash common.Hash) (*types.Receipt, error)
	GetTransaction(pool PoolCore, bcStore store.BlockchainStore, txHash common.Hash) (*types.Transaction, *BlockIndex, error)
}

func GetAPIs(apiBackend Backend) []rpc.API {
	return []rpc.API{
		{
			Namespace: "scdo",
			Version:   "1.0",
			Service:   NewPublicSeeleAPI(apiBackend),
			Public:    true,
		},
		{
			Namespace: "txpool",
			Version:   "1.0",
			Service:   NewTransactionPoolAPI(apiBackend),
			Public:    true,
		},
		{
			Namespace: "network",
			Version:   "1.0",
			Service:   NewPrivateNetworkAPI(apiBackend),
			Public:    true,
		},
	}
}

// MinerInfo miner simple info
type GetMinerInfo struct {
	Coinbase           common.Address
	CurrentBlockHeight uint64
	HeaderHash         common.Hash
	Shard              uint
	MinerStatus        string
	Version            string
	BlockAge           *big.Int
	PeerCnt            string
}

type GetMinerInfo2 struct {
	Coinbase           string
	CurrentBlockHeight uint64
	HeaderHash         common.Hash
	Shard              uint
	MinerStatus        string
	Version            string
	BlockAge           *big.Int
	PeerCnt            string
}

// GetBalanceResponse response param for GetBalance api
type GetBalanceResponse struct {
	Account common.Address
	Balance *big.Int
}

// GetLogsResponse response param for GetLogs api
type GetLogsResponse struct {
	*types.Log
	Txhash   common.Hash
	LogIndex uint
	Args     interface{} `json:"data"`
}

type PoolCore interface {
	AddTransaction(tx *types.Transaction) error
	GetTransaction(txHash common.Hash) *types.Transaction
}

type Pool interface {
	PoolCore
	GetTransactions(processing, pending bool) []*types.Transaction
	GetTxCount() int
}

type Chain interface {
	CurrentHeader() *types.BlockHeader
	GetCurrentState() (*state.Statedb, error)
	GetState(blockHash common.Hash) (*state.Statedb, error)
	GetStore() store.BlockchainStore
}

type Protocol interface {
	SendDifferentShardTx(tx *types.Transaction, shard uint)
	GetProtocolVersion() (uint, error)
}
