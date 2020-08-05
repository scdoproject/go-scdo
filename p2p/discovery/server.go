/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package discovery

import (
	"net"

	"github.com/scdoproject/go-scdo/common"
)

// StartService start node udp service
func StartService(nodeDir string, myID common.Address, myAddr *net.UDPAddr, bootstrap []*Node, shard uint) (*Database, *UDP) {
	udp := newUDP(myID, myAddr, shard)
	if bootstrap != nil {
		udp.trustNodes = bootstrap
	}
	udp.loadNodes(nodeDir)
	udp.loadBlockList(nodeDir)
	udp.StartServe(nodeDir)

	return udp.db, &UDP{udp: *udp}
}
