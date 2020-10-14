/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package scdo

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/scdoproject/go-scdo/accounts/abi"
	api2 "github.com/scdoproject/go-scdo/api"
	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/common/errors"
	"github.com/scdoproject/go-scdo/common/hexutil"
	"github.com/scdoproject/go-scdo/core"
	"github.com/scdoproject/go-scdo/core/state"
	"github.com/scdoproject/go-scdo/core/types"
	"github.com/scdoproject/go-scdo/crypto"
)

// PublicScdoAPI provides an API to access full node-related information.
type PublicScdoAPI struct {
	s *ScdoService
}

const maxSizeLimit = 64

// NewPublicSeeleAPI creates a new PublicScdoAPI object for rpc service.
func NewPublicSeeleAPI(s *ScdoService) *PublicScdoAPI {
	return &PublicScdoAPI{s}
}

// EstimateGas returns an estimate of the amount of gas needed to execute the
// given transaction against the current pending block.
func (api *PublicScdoAPI) EstimateGas(tx *types.Transaction) (uint64, error) {
	// Get the block by block height, if the height is less than zero, get the current block.
	block, err := getBlock(api.s.chain, -1)
	if err != nil {
		return 0, err
	}

	// Get the statedb by the given block height
	statedb, err := state.NewStatedb(block.Header.StateHash, api.s.accountStateDB)
	if err != nil {
		return 0, err
	}

	coinbase := api.s.miner.GetCoinbase()
	// Get the transaction receipt, and the fee give to the miner coinbase
	receipt, err := api.s.chain.ApplyTransaction(tx, 0, coinbase, statedb, block.Header)
	if err != nil {
		return 0, err
	}
	if receipt.Failed {
		return 0, errors.New(string(receipt.Result))
	}
	return receipt.UsedGas, nil
}

// GetInfo gets the account address that mining rewards will be send to.
func (api *PublicScdoAPI) GetInfo() (api2.GetMinerInfo2, error) {
	block := api.s.chain.CurrentBlock()

	var status string
	if api.s.miner.IsMining() {
		status = "Running"
	} else {
		status = "Stopped"
	}
	p1 := api.s.scdoProtocol.peerSet.getPeerCountByShard(1)
	p2 := api.s.scdoProtocol.peerSet.getPeerCountByShard(2)
	p3 := api.s.scdoProtocol.peerSet.getPeerCountByShard(3)
	p4 := api.s.scdoProtocol.peerSet.getPeerCountByShard(4)
	p0 := p1 + p2 + p3 + p4
	peers := fmt.Sprintf("%d (%d %d %d %d)", p0, p1, p2, p3, p4)
	return api2.GetMinerInfo2{
		Coinbase:           api.s.miner.GetCoinbase().String(),
		CurrentBlockHeight: block.Header.Height,
		HeaderHash:         block.HeaderHash,
		Shard:              common.LocalShardNumber,
		MinerStatus:        status,
		Version:            common.ScdoNodeVersion,
		BlockAge:           new(big.Int).Sub(big.NewInt(time.Now().Unix()), block.Header.CreateTimestamp),
		PeerCnt:            peers,
	}, nil
}

// Call is to execute a given transaction on a statedb of a given block height.
// It does not affect this statedb and blockchain and is useful for executing and retrieve values.
func (api *PublicScdoAPI) Call(contract, payload string, height int64) (map[string]interface{}, error) {
	contractAddr, err := common.HexToAddress(contract)
	if err != nil {
		return nil, fmt.Errorf("invalid contract address: %s", err)
	}

	msg, err := hexutil.HexToBytes(payload)
	if err != nil {
		return nil, fmt.Errorf("invalid payload, %s", err)
	}

	// Get the block by block height, if the height is less than zero, get the current block.
	block, err := getBlock(api.s.chain, height)
	if err != nil {
		return nil, err
	}

	// Get the statedb by the given block height
	statedb, err := state.NewStatedb(block.Header.StateHash, api.s.accountStateDB)
	if err != nil {
		return nil, err
	}

	coinbase := api.s.miner.GetCoinbase()
	from := crypto.MustGenerateShardAddress(coinbase.Shard())
	statedb.CreateAccount(*from)
	statedb.SetBalance(*from, common.ScdoToWen)

	amount, price, nonce := big.NewInt(0), big.NewInt(1), uint64(1)
	// gasLimit = balance / fee
	gasLimit := common.ScdoToWen.Uint64()
	tx, err := types.NewMessageTransaction(*from, contractAddr, amount, price, gasLimit, nonce, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %s", err)
	}

	// Get the transaction receipt, and the fee give to the miner coinbase
	receipt, err := api.s.chain.ApplyTransaction(tx, 0, coinbase, statedb, block.Header)
	if err != nil {
		return nil, err
	}

	// Format the receipt
	result, err := api2.PrintableReceipt(receipt)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetLogs Get the logs that satisfies the condition in the block by height and filter
func (api *PublicScdoAPI) GetLogs(height int64, contractAddress common.Address, abiJSON, eventName string) ([]api2.GetLogsResponse, error) {
	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, errors.NewStackedError(err, "get abi parser failed")
	}

	event, ok := parsed.Events[eventName]
	if !ok {
		return nil, fmt.Errorf("event name %v not found in ABI file", eventName)
	}

	topic := event.Id()

	// Do filter
	block, err := getBlock(api.s.chain, height)
	if err != nil {
		return nil, err
	}

	store := api.s.chain.GetStore()
	receipts, err := store.GetReceiptsByBlockHash(block.HeaderHash)
	if err != nil {
		return nil, err
	}

	logs := make([]api2.GetLogsResponse, 0)
	for _, receipt := range receipts {
		for logIndex, log := range receipt.Logs {
			// Matches contract address
			if !contractAddress.Equal(log.Address) {
				continue
			}

			// Matches topics
			// Because of the topics is always only one
			if len(log.Topics) < 1 || !topic.Equal(log.Topics[0]) {
				continue
			}

			data, err := event.Inputs.UnpackValues(log.Data)
			if err != nil {
				return nil, errors.NewStackedError(err, "failed to decode event arguments")
			}

			logs = append(logs, api2.GetLogsResponse{log, receipt.TxHash, uint(logIndex), data})
		}
	}

	return logs, nil
}

// getBlock returns block by height,when height is less than 0 the chain head is returned
func getBlock(chain *core.Blockchain, height int64) (*types.Block, error) {
	var block *types.Block
	if height < 0 {
		block = chain.CurrentBlock()
	} else {
		var err error
		block, err = chain.GetStore().GetBlockByHeight(uint64(height))
		if err != nil {
			return nil, err
		}
	}

	return block, nil
}

// GetShardNum gets the account shard number .
// if the address is valid, return the corresponding shard number, otherwise return 0
func (api *PublicScdoAPI) GetShardNum(account common.Address) (uint, error) {
	err := account.Validate()
	if err == nil {
		return account.Shard(), nil
	} else {
		return 0, err
	}
}

// GetWork get the work needed to be done
func (api *PublicScdoAPI) GetWork() map[string]interface{} {
	return api.s.miner.GetWork()
}

func (api *PublicScdoAPI) GetCurrentWorkHeader() map[string]interface{} {
	return api.s.miner.GetCurrentWorkHeader()
}

func (api *PublicScdoAPI) SubmitNonce(height uint64, nonce uint64) error {
	return api.s.miner.SubmitWork(height, nonce)
}
