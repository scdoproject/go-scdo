/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package light

import (
	rand2 "math/rand"
	"time"

	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/core/types"
	"github.com/scdoproject/go-scdo/event"
	"github.com/scdoproject/go-scdo/log"
	"github.com/scdoproject/go-scdo/node"
	"github.com/scdoproject/go-scdo/p2p"
	"github.com/scdoproject/go-scdo/rpc"
	"github.com/scdoproject/go-scdo/scdo"
)

// ServiceServer implements light server service.
type ServiceServer struct {
	p2pServer     *p2p.Server
	scdoProtocol *LightProtocol
	log           *log.ScdoLog
	shard         uint
}

// NewServiceServer create ServiceServer
func NewServiceServer(service *scdo.ScdoService, conf *node.Config, log *log.ScdoLog, shard uint) (*ServiceServer, error) {
	scdoProtocol, err := NewLightProtocol(conf.P2PConfig.NetworkID, service.TxPool(), service.DebtPool(), service.BlockChain(), true, nil, log, shard)
	if err != nil {
		return nil, err
	}

	s := &ServiceServer{
		log:           log,
		scdoProtocol: scdoProtocol,
	}

	rand2.Seed(time.Now().UnixNano())
	s.log.Info("Light server started")
	return s, nil
}

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *ServiceServer) Protocols() (protos []p2p.Protocol) {
	return append(protos, s.scdoProtocol.Protocol)
}

// Start implements node.Service, starting goroutines needed by ServiceServer.
func (s *ServiceServer) Start(srvr *p2p.Server) error {
	s.p2pServer = srvr

	s.scdoProtocol.Start()
	go s.scdoProtocol.blockLoop()
	return nil
}

// Stop implements node.Service, terminating all internal goroutines.
func (s *ServiceServer) Stop() error {
	s.scdoProtocol.Stop()
	return nil
}

// APIs implements node.Service, returning the collection of RPC services the scdo package offers.
func (s *ServiceServer) APIs() (apis []rpc.API) {
	return
}

func (pm *LightProtocol) chainHeaderChanged(e event.Event) {
	newBlock := e.(*types.Block)
	if newBlock == nil || newBlock.HeaderHash.IsEmpty() {
		return
	}

	pm.chainHeaderChangeCh <- newBlock.HeaderHash
}

// as light node server, when this node's chain header has changed, broadcast it to all light node client peers
func (pm *LightProtocol) blockLoop() {
	pm.wg.Add(1)
	defer pm.wg.Done()
	pm.chainHeaderChangeCh = make(chan common.Hash, 1)
	event.ChainHeaderChangedEventMananger.AddAsyncListener(pm.chainHeaderChanged)
needQuit:
	for {
		select {
		case <-pm.chainHeaderChangeCh:
			magic := rand2.Uint32()
			peers := pm.peerSet.getPeers()
			for _, p := range peers {
				if p != nil {
					if lastTime, ok := pm.peerSet.peerLastAnnounceTimeMap[p]; ok && (time.Now().Unix()-lastTime < 5) {
						pm.log.Debug("blockLoop sendAnnounce cancelled,magic:%d,peer:%s", magic, p.peerStrID)
						continue
					}
					pm.peerSet.peerLastAnnounceTimeMap[p] = time.Now().Unix()
					pm.log.Debug("blockLoop sendAnnounce,magic:%d,peer:%s", magic, p.peerStrID)
					err := p.sendAnnounce(magic, uint64(0), uint64(0))
					if err != nil {
						pm.log.Debug("blockLoop sendAnnounce err=%s", err)
					}
				}
			}

			pm.log.Debug("blockLoop head changed. ")

		case <-pm.quitCh:
			break needQuit
		}
	}

	event.ChainHeaderChangedEventMananger.RemoveListener(pm.chainHeaderChanged)
	close(pm.chainHeaderChangeCh)
}
