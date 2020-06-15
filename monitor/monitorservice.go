/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package monitor

import (
	"github.com/seelecredo/go-seelecredo/log"
	"github.com/seelecredo/go-seelecredo/node"
	"github.com/seelecredo/go-seelecredo/p2p"
	rpc "github.com/seelecredo/go-seelecredo/rpc"
	"github.com/seelecredo/go-seelecredo/seeleCredo"
)

// MonitorService implements some rpc interfaces provided by a monitor server
type MonitorService struct {
	p2pServer  *p2p.Server                   // Peer-to-Peer server infos
	seeleCredo *seeleCredo.ScdoService // seeleCredo full node service
	slcNode    *node.Node                    // seeleCredo node
	log        *log.ScdoLog

	rpcAddr string // listening port
	name    string // name displayed on the moitor
	node    string // node name
	version string // version
}

// NewMonitorService returns a MonitorService instance
func NewMonitorService(slcService *seeleCredo.ScdoService, slcNode *node.Node, conf *node.Config, slclog *log.ScdoLog, name string) (*MonitorService, error) {
	return &MonitorService{
		seeleCredo: slcService,
		slcNode:    slcNode,
		log:        slclog,
		name:       name,
		rpcAddr:    conf.BasicConfig.RPCAddr,
		node:       conf.BasicConfig.Name,
		version:    conf.BasicConfig.Version,
	}, nil
}

// Protocols implements node.Service, return nil as it dosn't use the p2p service
func (s *MonitorService) Protocols() []p2p.Protocol { return nil }

// Start implements node.Service, starting goroutines needed by ScdoService.
func (s *MonitorService) Start(srvr *p2p.Server) error {
	s.p2pServer = srvr

	s.log.Info("monitor rpc service start")

	return nil
}

// Stop implements node.Service, terminating all internal goroutines.
func (s *MonitorService) Stop() error {

	return nil
}

// APIs implements node.Service, returning the collection of RPC services the seeleCredo package offers.
func (s *MonitorService) APIs() (apis []rpc.API) {
	return append(apis, []rpc.API{
		{
			Namespace: "monitor",
			Version:   "1.0",
			Service:   NewPublicMonitorAPI(s),
			Public:    true,
		},
	}...)
}
