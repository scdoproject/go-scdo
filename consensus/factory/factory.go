/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package factory

import (
	"crypto/ecdsa"
	"fmt"
	"path/filepath"

	"github.com/seeledevteam/slc/common"
	"github.com/seeledevteam/slc/common/errors"
	"github.com/seeledevteam/slc/consensus"
	"github.com/seeledevteam/slc/consensus/ethash"
	"github.com/seeledevteam/slc/consensus/istanbul"
	"github.com/seeledevteam/slc/consensus/istanbul/backend"
	"github.com/seeledevteam/slc/consensus/pow"
	"github.com/seeledevteam/slc/consensus/spow"
	"github.com/seeledevteam/slc/database/leveldb"
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
