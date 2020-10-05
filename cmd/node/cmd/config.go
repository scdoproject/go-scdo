/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package cmd

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/scdoproject/go-scdo/cmd/util"
	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/core"
	"github.com/scdoproject/go-scdo/crypto"
	"github.com/scdoproject/go-scdo/log/comm"
	"github.com/scdoproject/go-scdo/node"
	"github.com/scdoproject/go-scdo/p2p"
)

// GetConfigFromFile unmarshals the config from the given file
func GetConfigFromFile(filepath string) (*util.Config, error) {
	var config util.Config
	buff, err := ioutil.ReadFile(filepath)
	if err != nil {
		return &config, err
	}

	err = json.Unmarshal(buff, &config)
	return &config, err
}

// Cast cast RPC address to 0.0.0.0
// miner mehtods already have security-defence setting, 0.0.0.0 is ok (after mainnet matures and becomes stable, we can switch to 127.0.0.1)
func Cast(conf *node.Config) {
	endpoint := conf.BasicConfig.RPCAddr
	pos := strings.LastIndex(endpoint, ":")
	port := endpoint[pos+1:]
	endpoint = "0.0.0.0:" + port
	conf.BasicConfig.RPCAddr = endpoint
}

// LoadConfigFromFile gets node config from the given file
func LoadConfigFromFile(configFile string, accounts string, poolAccounts string) (*node.Config, error) {
	cmdConfig, err := GetConfigFromFile(configFile)
	if err != nil {
		return nil, err
	}

	if cmdConfig.GenesisConfig.CreateTimestamp == nil {
		return nil, errors.New("Failed to get genesis timestamp")
	}
	cmdConfig.GenesisConfig.Accounts, err = LoadAccountConfig(accounts)
	if err != nil {
		return nil, err
	}

	config := CopyConfig(cmdConfig)
	convertIPCServerPath(cmdConfig, config)

	config.P2PConfig, err = GetP2pConfig(cmdConfig)
	if err != nil {
		return config, err
	}

	if len(config.BasicConfig.Coinbase) > 0 {
		// fmt.Println(config.BasicConfig.Coinbase)
		config.ScdoConfig.Coinbase = common.HexMustToAddres(config.BasicConfig.Coinbase)
	}

	if len(config.BasicConfig.PrivateKey) > 0 {
		config.ScdoConfig.CoinbasePrivateKey, err = crypto.LoadECDSAFromString(config.BasicConfig.PrivateKey)
		if err != nil {
			return config, err
		}
	}

	if len(poolAccounts) > 0 {
		config.ScdoConfig.CoinbaseList, err = LoadPoolAccountConfig(poolAccounts)
		if err != nil {
			return nil, err
		}
	}

	config.ScdoConfig.TxConf = *core.DefaultTxPoolConfig()
	config.ScdoConfig.GenesisConfig = cmdConfig.GenesisConfig
	comm.LogConfiguration.PrintLog = config.LogConfig.PrintLog
	comm.LogConfiguration.IsDebug = config.LogConfig.IsDebug
	comm.LogConfiguration.DataDir = config.BasicConfig.DataDir
	config.BasicConfig.DataDir = filepath.Join(common.GetDefaultDataFolder(), config.BasicConfig.DataDir)
	return config, nil
}

// convertIPCServerPath convert the config to the real path
func convertIPCServerPath(cmdConfig *util.Config, config *node.Config) {
	if cmdConfig.Ipcconfig.PipeName == "" {
		config.IpcConfig.PipeName = common.GetDefaultIPCPath()
	} else if runtime.GOOS == "windows" {
		config.IpcConfig.PipeName = common.WindowsPipeDir + cmdConfig.Ipcconfig.PipeName
	} else {
		config.IpcConfig.PipeName = filepath.Join(common.GetDefaultDataFolder(), cmdConfig.Ipcconfig.PipeName)
	}
}

// CopyConfig copy Config from the given config
func CopyConfig(cmdConfig *util.Config) *node.Config {
	config := &node.Config{
		BasicConfig:    cmdConfig.BasicConfig,
		LogConfig:      cmdConfig.LogConfig,
		HTTPServer:     cmdConfig.HTTPServer,
		WSServerConfig: cmdConfig.WSServerConfig,
		P2PConfig:      cmdConfig.P2PConfig,
		ScdoConfig:     node.ScdoConfig{},
		MetricsConfig:  cmdConfig.MetricsConfig,
	}
	return config
}

// GetP2pConfig get P2PConfig from the given config
func GetP2pConfig(cmdConfig *util.Config) (p2p.Config, error) {
	if cmdConfig.P2PConfig.PrivateKey == nil {
		key, err := crypto.LoadECDSAFromString(cmdConfig.P2PConfig.SubPrivateKey) // GetP2pConfigPrivateKey get privateKey from the given config
		if err != nil {
			return cmdConfig.P2PConfig, err
		}
		cmdConfig.P2PConfig.PrivateKey = key
	}
	return cmdConfig.P2PConfig, nil
}

func LoadAccountConfig(account string) (map[common.Address]*big.Int, error) {
	result := make(map[common.Address]*big.Int)
	if account == "" {
		return result, nil
	}

	buff, err := ioutil.ReadFile(account)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(buff, &result)
	return result, err
}

func LoadPoolAccountConfig(account string) ([]common.Address, error) {
	addrMap := make(map[common.Address]*big.Int)
	var result []common.Address
	if account == "" {
		return result, nil
	}

	buff, err := ioutil.ReadFile(account)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(buff, &addrMap)

	for addr, _ := range addrMap {
		result = append(result, addr)
	}
	return result, err
}
