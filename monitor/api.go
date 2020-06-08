/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package monitor

import (
	"errors"
	"runtime"

	"github.com/seeledevteam/slc/common"
)

// error infos
var (
	ErrBlockchainInfoFailed = errors.New("getting blockchain info failed")
	ErrMinerInfoFailed      = errors.New("getting miner info failed")
	ErrNodeInfoFailed       = errors.New("getting node info failed")
	ErrP2PServerInfoFailed  = errors.New("getting p2p server info failed")
)

// PublicMonitorAPI provides an API to the monitor service
type PublicMonitorAPI struct {
	s *MonitorService
}

// NewPublicMonitorAPI create new PublicMonitorAPI
func NewPublicMonitorAPI(s *MonitorService) *PublicMonitorAPI {
	return &PublicMonitorAPI{s}
}

// NodeInfo returns the NodeInfo struct of the local node
func (api *PublicMonitorAPI) NodeInfo() (NodeInfo, error) {
	return NodeInfo{
		Name:       api.s.name,
		Node:       api.s.node,
		Port:       0, //api.s.p2pServer.ListenAddr,
		NetVersion: api.s.seeleCredo.NetVersion(),
		Protocol:   "1.0",
		API:        "No",
		Os:         runtime.GOOS,
		OsVer:      runtime.GOARCH,
		Client:     api.s.version,
		History:    true,
		Shard:      common.LocalShardNumber,
	}, nil
}

// NodeStats returns the information about the local node.
func (api *PublicMonitorAPI) NodeStats() (*NodeStats, error) {
	if api.s.p2pServer == nil {
		return nil, ErrP2PServerInfoFailed
	}

	if api.s.slcNode == nil {
		return nil, ErrNodeInfoFailed
	}

	if api.s.seeleCredo.Miner() == nil {
		return nil, ErrMinerInfoFailed
	}

	mining := api.s.seeleCredo.Miner().IsMining()

	return &NodeStats{
		Active:  true,
		Syncing: true,
		Mining:  mining,
		Peers:   api.s.p2pServer.PeerCount(),
	}, nil
}
