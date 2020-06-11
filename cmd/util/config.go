/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package util

import (
	"github.com/seelecredoteam/go-seelecredo/core"
	"github.com/seelecredoteam/go-seelecredo/log/comm"
	"github.com/seelecredoteam/go-seelecredo/metrics"
	"github.com/seelecredoteam/go-seelecredo/node"
	"github.com/seelecredoteam/go-seelecredo/p2p"
)

// Config is the Configuration of node
type Config struct {
	//Config is the Configuration of log
	LogConfig comm.LogConfig `json:"log"`

	// basic config for Node
	BasicConfig node.BasicConfig `json:"basic"`

	// The configuration of p2p network
	P2PConfig p2p.Config `json:"p2p"`

	// HttpServer config for http server
	HTTPServer node.HTTPServer `json:"httpServer"`

	// The configuration of websocket rpc service
	WSServerConfig node.WSServerConfig `json:"wsserver"`

	// The configuration of ipc rpc service
	Ipcconfig node.IpcConfig `json:"ipcconfig"`

	// metrics config info
	MetricsConfig *metrics.Config `json:"metrics"`

	// genesis config info
	GenesisConfig core.GenesisInfo `json:"genesis"`
}
