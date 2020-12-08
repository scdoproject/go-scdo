/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package zpow

import (
	"encoding/binary"
	"math"
	"math/big"
	"math/rand"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rcrowley/go-metrics"
	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/consensus"
	"github.com/scdoproject/go-scdo/consensus/scdorand"
	"github.com/scdoproject/go-scdo/consensus/utils"
	"github.com/scdoproject/go-scdo/core/types"
	"github.com/scdoproject/go-scdo/log"
	"github.com/scdoproject/go-scdo/rpc"
	"gonum.org/v1/gonum/mat"
)

var (
	// bound of the determinant
	maxDet30x30 = new(big.Int).Mul(big.NewInt(2), new(big.Int).Exp(big.NewInt(10), big.NewInt(30), big.NewInt(0)))
	matrixDim   = int(30)
	multiplier  = big.NewInt(3000000000)
)

// Engine provides the consensus operations based on ZPOW.
type ZpowEngine struct {
	threads int
	log     *log.ScdoLog
	detrate metrics.Meter
	lock    sync.Mutex
}

func NewZpowEngine(threads int) *ZpowEngine {

	return &ZpowEngine{
		threads: threads,
		log:     log.GetLogger("zpow_engine"),
		detrate: metrics.NewMeter(),
	}
}

func (engine *ZpowEngine) SetThreads(threads int) {
	if threads <= 0 {
		engine.threads = runtime.NumCPU()
	} else {
		engine.threads = threads
	}
}

func (engine *ZpowEngine) APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{
		{
			Namespace: "miner",
			Version:   "1.0",
			Service:   &API{engine},
			Public:    true,
		},
	}
}

func (engine *ZpowEngine) Prepare(reader consensus.ChainReader, header *types.BlockHeader) error {
	parent := reader.GetHeaderByHash(header.PreviousBlockHash)
	if parent == nil {
		return consensus.ErrBlockInvalidParentHash
	}

	header.Difficulty = utils.GetDifficult(header.CreateTimestamp.Uint64(), parent)

	return nil
}

func (engine *ZpowEngine) Seal(reader consensus.ChainReader, block *types.Block, stop <-chan struct{}, results chan<- *types.Block) error {
	threads := engine.threads

	var step uint64
	var seed uint64
	if threads != 0 {
		step = math.MaxUint64 / uint64(threads)
	}

	var isNonceFound int32
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	once := &sync.Once{}
	for i := 0; i < threads; i++ {
		if threads == 1 {
			seed = r.Uint64()
		} else {
			seed = uint64(r.Int63n(int64(step)))
		}
		tSeed := seed + uint64(i)*step
		var min uint64
		var max uint64
		min = uint64(i) * step

		if i != threads-1 {
			max = min + step - 1
		} else {
			max = math.MaxUint64
		}

		go func(tseed uint64, tmin uint64, tmax uint64) {
			engine.StartMining(block, tseed, tmin, tmax, results, stop, &isNonceFound, once, engine.detrate, engine.log)
		}(tSeed, min, max)
	}

	return nil
}

func (engine *ZpowEngine) StartMining(block *types.Block, seed uint64, min uint64, max uint64, result chan<- *types.Block, abort <-chan struct{},
	isNonceFound *int32, once *sync.Once, detrate metrics.Meter, log *log.ScdoLog) {
	var nonce = seed
	var caltimes = int64(0)
	target := new(big.Float).SetInt(getMiningTarget(block.Header.Difficulty))
	header := block.Header.Clone()
	dim := matrixDim

miner:
	for {
		select {
		case <-abort:
			logAbort(log)
			detrate.Mark(caltimes)
			break miner

		default:
			if atomic.LoadInt32(isNonceFound) != 0 {
				log.Debug("exit mining as nonce is found by other threads")
				break miner
			}

			caltimes++
			detrate.Mark(1)
			if caltimes == 0X7FFFFFFFFFFFFFFF {
				caltimes = 0
			}

			header.Witness = []byte(strconv.FormatUint(nonce, 10))
			hash := header.Hash()

			// generate matrix
			matrix := generateRandomMat(hash, dim, header.Height)

			// compute matrix det
			res := mat.Det(matrix)
			restBig := big.NewFloat(res)
			// found
			if restBig.Cmp(target) >= 0 {
				once.Do(func() {
					block.Header = header
					block.HeaderHash = hash

					select {
					case <-abort:
						logAbort(log)
					case result <- block:
						atomic.StoreInt32(isNonceFound, 1)
						log.Debug("found det:%e", restBig)
						log.Debug("target:%e", target)
						log.Debug("times2try:%d", caltimes)
					}
				})
				break miner
			}

			// when nonce reached max, nonce traverses in [min, seed-1]
			if nonce == max {
				nonce = min
			}
			// outage
			if nonce == seed-1 {
				select {
				case <-abort:
					logAbort(log)
				case result <- nil:
					log.Warn("nonce finding outage")
				}

				break miner
			}
			nonce++
		}
	}
}

// ValidateHeader validates the specified header and returns error if validation failed.
func (engine *ZpowEngine) VerifyHeader(reader consensus.ChainReader, header *types.BlockHeader) error {
	parent := reader.GetHeaderByHash(header.PreviousBlockHash)
	if parent == nil {
		engine.log.Info("invalid parent hash: %v", header.PreviousBlockHash)
		return consensus.ErrBlockInvalidParentHash
	}

	if err := utils.VerifyHeaderCommon(header, parent); err != nil {
		return err
	}

	if err := engine.verifyTarget(header); err != nil {
		return err
	}

	return nil
}

// block verification
func (engine *ZpowEngine) verifyTarget(header *types.BlockHeader) error {
	dim := matrixDim
	NewHeader := header.Clone()
	hash := NewHeader.Hash()

	// generate matrix
	matrix := generateRandomMat(hash, dim, header.Height)

	// compute matrix det
	res := mat.Det(matrix)
	restBig := big.NewFloat(res)
	target := new(big.Float).SetInt(getMiningTarget(header.Difficulty))
	if restBig.Cmp(target) < 0 {
		return consensus.ErrBlockNonceInvalid
	}
	return nil
}

// getMiningTarget returns the mining target for the specified difficulty.
func getMiningTarget(difficulty *big.Int) *big.Int {
	target := new(big.Int).Mul(difficulty, multiplier)
	if target.Cmp(maxDet30x30) > 0 {
		return maxDet30x30
	}
	return target
}

// logAbort logs the info that nonce finding is aborted
func logAbort(log *log.ScdoLog) {
	log.Info("nonce finding aborted")
}

func generateRandomMat32(hash common.Hash, dim int) *mat.Dense {
	matrix := mat.NewDense(dim, dim, nil)
	hashBytes := hash.Bytes()
	var hashSeed [4]int64
	curNum := int64(0)
	hashSeed[0] = bytesToInt64(hashBytes[:8])
	hashSeed[1] = bytesToInt64(hashBytes[8:16])
	hashSeed[2] = bytesToInt64(hashBytes[16:24])
	hashSeed[3] = bytesToInt64(hashBytes[24:32])
	for i := 0; i < dim; i++ {
		curNum ^= hashSeed[i%4]
		r := rand.New(rand.NewSource(curNum))
		for j := 0; j < dim; j++ {
			curNum = r.Int63n(1<<60 - 1)
			matrix.Set(i, j, float64(r.Int63n(2)))
		}
	}
	return matrix
}

func bytesToInt64(buf []byte) int64 {
	return int64(binary.BigEndian.Uint64(buf))
}

func generateRandomMat(hash common.Hash, dim int, height uint64) *mat.Dense {
	matrix := mat.NewDense(dim, dim, nil)
	hashBytes := hash.Bytes()
	var hashSeed [4]int64
	curNum := int64(0)
	hashSeed[0] = bytesToInt64(hashBytes[:8])
	hashSeed[1] = bytesToInt64(hashBytes[8:16])
	hashSeed[2] = bytesToInt64(hashBytes[16:24])
	hashSeed[3] = bytesToInt64(hashBytes[24:32])
	for i := 0; i < dim; i++ {
		curNum ^= hashSeed[i%4]
		var randObj *scdorand.RandObj
		if height >= common.EmeryForkHeight {
			randObj = scdorand.NewRandObj(scdorand.NewSource_EmeryFork(curNum))
		} else {
			randObj = scdorand.NewRandObj(scdorand.NewSource(curNum))
		}
		for j := 0; j < dim; j++ {
			curNum = randObj.Int63n(1<<63 - 1)
			matrix.Set(i, j, float64(randObj.Int63n(3)))
		}
	}
	return matrix
}
