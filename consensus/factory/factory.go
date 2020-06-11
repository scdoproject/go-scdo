/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package factory

import (
	"crypto/ecdsa"
	"fmt"
	"path/filepath"

	"github.com/seelecredo/go-seelecredo/common"
	"github.com/seelecredo/go-seelecredo/common/errors"
	"github.com/seelecredo/go-seelecredo/consensus"
	"github.com/seelecredo/go-seelecredo/consensus/ethash"
	"github.com/seelecredo/go-seelecredo/consensus/istanbul"
	"github.com/seelecredo/go-seelecredo/consensus/istanbul/backend"
	"github.com/seelecredo/go-seelecredo/consensus/pow"
	"github.com/seelecredo/go-seelecredo/consensus/spow"
	"github.com/seelecredo/go-seelecredo/database/leveldb"
)

// GetConsensusEngine get consensus engine according to miner algorithm name
// WARNING: engine may be a heavy instance. we should have as less as possible in our process.
func GetConsensusEngine(minerAlgorithm string, folder string, percentage int) (consensus.Engine, error) {
	var minerEngine consensus.Engine
	if minerAlgorithm == common.EthashAlgorithm {
		minerEngine = ethash.New(ethash.GetDefaultConfig(), nil, false)
	} else if minerAlgorithm == common.Sha256Algorithm {
		minerEngine = pow.NewEngine(1)
	} else if minerAlgorithm == common.SpowAlgorithm {
		minerEngine = spow.NewSpowEngine(1, folder, percentage)
	} else {
		return nil, fmt.Errorf("unknown miner algorithm")
	}

	return minerEngine, nil
}

func GetBFTEngine(privateKey *ecdsa.PrivateKey, folder string) (consensus.Engine, error) {
	path := filepath.Join(folder, common.BFTDataFolder)
	db, err := leveldb.NewLevelDB(path)
	if err != nil {
		return nil, errors.NewStackedError(err, "create bft folder failed")
	}

	return backend.New(istanbul.DefaultConfig, privateKey, db), nil
}

func MustGetConsensusEngine(minerAlgorithm string) consensus.Engine {
	engine, err := GetConsensusEngine(minerAlgorithm, "temp", 10)
	if err != nil {
		panic(err)
	}

	return engine
}
