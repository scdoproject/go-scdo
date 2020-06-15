/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package discovery

import (
	"net"
)

func getUDPConn(addr *net.UDPAddr) (*net.UDPConn, error) {
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
