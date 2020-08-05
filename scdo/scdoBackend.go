package scdo

import (
	"math/big"

	"github.com/scdoproject/go-scdo/api"
	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/core/store"
	"github.com/scdoproject/go-scdo/core/types"
	"github.com/scdoproject/go-scdo/log"
	"github.com/scdoproject/go-scdo/p2p"
	"github.com/scdoproject/go-scdo/p2p/discovery"
	downloader "github.com/scdoproject/go-scdo/scdo/download"
)

type ScdoBackend struct {
	s *ScdoService
}

// NewScdoBackend backend
func NewScdoBackend(s *ScdoService) *ScdoBackend {
	return &ScdoBackend{s}
}

// TxPoolBackend tx pool
func (sd *ScdoBackend) TxPoolBackend() api.Pool { return sd.s.txPool }

// GetNetVersion net version
func (sd *ScdoBackend) GetNetVersion() string { return sd.s.netVersion }

// GetNetWorkID net id
func (sd *ScdoBackend) GetNetWorkID() string { return sd.s.networkID }

// GetP2pServer p2p server
func (sd *ScdoBackend) GetP2pServer() *p2p.Server { return sd.s.p2pServer }

// GetUDPServer UDP server
func (sd *ScdoBackend) GetUDPerver() *discovery.UDP {
	return sd.s.udpServer
}

// ChainBackend block chain db
func (sd *ScdoBackend) ChainBackend() api.Chain { return sd.s.chain }

// Log return log pointer
func (sd *ScdoBackend) Log() *log.ScdoLog { return sd.s.log }

// IsSyncing check status
func (sd *ScdoBackend) IsSyncing() bool {
	scdoserviceAPI := sd.s.APIs()[5]
	d := scdoserviceAPI.Service.(downloader.PrivatedownloaderAPI)

	return d.IsSyncing()
}

// ProtocolBackend return protocol
func (sd *ScdoBackend) ProtocolBackend() api.Protocol { return sd.s.scdoProtocol }

// GetBlock returns the requested block by hash or height
func (sd *ScdoBackend) GetBlock(hash common.Hash, height int64) (*types.Block, error) {
	var block *types.Block
	var err error
	if !hash.IsEmpty() {
		store := sd.s.chain.GetStore()
		block, err = store.GetBlock(hash)
		if err != nil {
			return nil, err
		}
	} else {
		if height < 0 {
			header := sd.s.chain.CurrentHeader()
			block, err = sd.s.chain.GetStore().GetBlockByHeight(header.Height)
		} else {
			block, err = sd.s.chain.GetStore().GetBlockByHeight(uint64(height))
		}
		if err != nil {
			return nil, err
		}
	}

	return block, nil
}

// GetBlockTotalDifficulty return total difficulty
func (sd *ScdoBackend) GetBlockTotalDifficulty(hash common.Hash) (*big.Int, error) {
	store := sd.s.chain.GetStore()
	return store.GetBlockTotalDifficulty(hash)
}

// GetReceiptByTxHash get receipt by transaction hash
func (sd *ScdoBackend) GetReceiptByTxHash(hash common.Hash) (*types.Receipt, error) {
	store := sd.s.chain.GetStore()
	receipt, err := store.GetReceiptByTxHash(hash)
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

// GetTransaction return tx
func (sd *ScdoBackend) GetTransaction(pool api.PoolCore, bcStore store.BlockchainStore, txHash common.Hash) (*types.Transaction, *api.BlockIndex, error) {
	return api.GetTransaction(pool, bcStore, txHash)
}
