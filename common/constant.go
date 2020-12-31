/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package common

import (
	"math/big"
	"os/user"
	"path/filepath"
	"runtime"
	"time"
)

const (

	// ScdoProtoName protoName of Scdo service
	ScdoProtoName = "scdo"

	// ScdoVersion Version number of Scdo protocol
	ScdoVersion uint = 1

	// ScdoNodeVersion for simpler display
	ScdoNodeVersion string = "Scdo_V1.0.0"

	// ShardCount represents the total number of shards.
	ShardCount = 4

	// ShardByte represents the number of bytes used for shard information, must be smaller than 8
	ShardByte = 1

	// MetricsRefreshTime is the time of metrics sleep 1 minute
	MetricsRefreshTime = time.Minute

	// CPUMetricsRefreshTime is the time of metrics monitor cpu
	CPUMetricsRefreshTime = time.Second

	// ConfirmedBlockNumber is the block number for confirmed a block, it should be more than 12 in product
	ConfirmedBlockNumber = 120

	ScdoForkHeight = 2979594

	// emery hard fork: update zpow consensus and evm
	EmeryForkHeight = ScdoForkHeight

	// ForkHeight after this height we change the content of block: hardFork
	ForkHeight = ScdoForkHeight

	// ForkHeight after this height we change the content of block: hardFork
	SecondForkHeight = ScdoForkHeight

	// ForkHeight after this height we change the validation of tx: hardFork
	ThirdForkHeight = ScdoForkHeight

	SmartContractNonceForkHeight = ScdoForkHeight

	// SmartContractNonceFixHeight fix smart contract nonce bug when user use setNonce
	SmartContractNonceFixHeight = ScdoForkHeight

	// LightChainDir lightchain data directory based on config.DataRoot
	LightChainDir = "/db/lightchain"

	// Sha256Algorithm miner algorithm sha256
	Sha256Algorithm = "sha256"

	// zpow miner algorithm
	ZpowAlgorithm = "zpow"

	// BFT mineralgorithm
	BFTEngine = "bft"

	// BFT data folder
	BFTDataFolder = "bftdata"

	// EVMStackLimit increase evm stack limit to 8192
	EVMStackLimit = 8192

	// BlockPackInterval it's an estimate time.
	BlockPackInterval = 15 * time.Second

	// Height: fix the issue caused by forking from collapse database
	HeightFloor = uint64(707989)
	HeightRoof  = uint64(707996)

	WindowsPipeDir = `\\.\pipe\`

	defaultPipeFile = `\scdo.ipc`
)

var (
	// tempFolder used to store temp file, such as log files
	tempFolder string

	// defaultDataFolder used to store persistent data info, such as the database and keystore
	defaultDataFolder string

	// defaultIPCPath used to store the ipc file
	defaultIPCPath string
)

// Common big integers often used
var (
	Big1   = big.NewInt(1)
	Big2   = big.NewInt(2)
	Big3   = big.NewInt(3)
	Big0   = big.NewInt(0)
	Big32  = big.NewInt(32)
	Big256 = big.NewInt(256)
	Big257 = big.NewInt(257)
)

// init initialize the paths to store data
func init() {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}

	tempFolder = filepath.Join(usr.HomeDir, "scdoTemp")

	defaultDataFolder = filepath.Join(usr.HomeDir, ".scdo")

	if runtime.GOOS == "windows" {
		defaultIPCPath = WindowsPipeDir + defaultPipeFile
	} else {
		defaultIPCPath = filepath.Join(defaultDataFolder, defaultPipeFile)
	}
}

// GetTempFolder gets the temp folder
func GetTempFolder() string {
	return tempFolder
}

// GetDefaultDataFolder gets the default data Folder
func GetDefaultDataFolder() string {
	return defaultDataFolder
}

// GetDefaultIPCPath gets the default IPC path
func GetDefaultIPCPath() string {
	return defaultIPCPath
}
