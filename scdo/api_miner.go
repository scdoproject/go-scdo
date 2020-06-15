/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package scdo

import (
	"errors"
	"fmt"

	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/core/types"
	"github.com/scdoproject/go-scdo/miner"
)

// PrivateMinerAPI provides an API to access miner information.
type PrivateMinerAPI struct {
	s *ScdoService
}

// NewPrivateMinerAPI creates a new PrivateMinerAPI object for miner rpc service.
func NewPrivateMinerAPI(s *ScdoService) *PrivateMinerAPI {
	return &PrivateMinerAPI{s}
}

// Start API is used to start the miner with the given number of threads.
func (api *PrivateMinerAPI) Start() (bool, error) {
	if api.s.miner.IsMining() {
		return true, miner.ErrMinerIsRunning
	}

	return true, api.s.miner.Start()
}

// Status API is used to view the miner's status.
func (api *PrivateMinerAPI) Status() (string, error) {
	if api.s.miner.IsMining() {
		return "Running", nil
	}

	return "Stopped", nil
}

// Stop API is used to stop the miner.
func (api *PrivateMinerAPI) Stop() (bool, error) {
	if !api.s.miner.IsMining() {
		return true, miner.ErrMinerIsStopped
	}
	api.s.miner.Stop()

	return true, nil
}

// SetThreads  API is used to set the number of threads.
func (api *PrivateMinerAPI) SetThreads(threads int) (bool, error) {
	if threads < 0 {
		return false, errors.New("threads should be greater than zero")
	}

	api.s.miner.SetThreads(threads)
	return true, nil
}

// SetCoinbase API is used to set the coinbase.
func (api *PrivateMinerAPI) SetCoinbase(coinbaseStr string) (bool, error) {
	coinbase, err := common.HexToAddress(coinbaseStr)
	if err != nil {
		return false, err
	}
	if !common.IsShardEnabled() {
		return false, fmt.Errorf("local shard number is invalid:[%v], it must greater than %v, less than %v", common.LocalShardNumber, common.UndefinedShardNumber, common.ShardCount)
	}
	if coinbase.Shard() != common.LocalShardNumber {
		return false, fmt.Errorf("invalid shard number: coinbase shard number is [%v], but local shard number is [%v]", coinbase.Shard(), common.LocalShardNumber)
	}
	api.s.miner.SetCoinbase(coinbase)

	return true, nil
}

// GetCoinbase API is used to get the coinbase.
func (api *PrivateMinerAPI) GetCoinbase() (common.Address, error) {
	return api.s.miner.GetCoinbase(), nil
}

// GetTask return the hash of the current block, the seedHash, and the boundary condition to be met (“target”).
// GetWork get the work needed to done
func (api *PrivateMinerAPI) GetWork() map[string]interface{} {
	return api.s.miner.GetWork()

}

func (api *PrivateMinerAPI) GetCurrentWorkHeader() (header *types.BlockHeader) {
	return api.s.miner.GetCurrentWorkHeader()
}

func (api *PrivateMinerAPI) GetTarget() string {
	return api.s.miner.GetTaskDifficulty().String()

}
