/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package miner

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/common/memory"
	"github.com/scdoproject/go-scdo/consensus"
	"github.com/scdoproject/go-scdo/core"
	"github.com/scdoproject/go-scdo/core/types"
	"github.com/scdoproject/go-scdo/event"
	"github.com/scdoproject/go-scdo/log"
)

var (
	// ErrMinerIsRunning is returned when miner is running
	ErrMinerIsRunning = errors.New("miner is running")

	// ErrMinerIsStopped is returned when miner is stopped
	ErrMinerIsStopped = errors.New("miner is stopped")

	// ErrNodeIsSyncing is returned when the node is syncing
	ErrNodeIsSyncing = errors.New("can not start miner when syncing")

	minerCount = 0
)

// ScdoBackend wraps all methods required for miner.
type ScdoBackend interface {
	TxPool() *core.TransactionPool
	BlockChain() *core.Blockchain
	DebtPool() *core.DebtPool
}

// Miner defines base elements of miner
type Miner struct {
	mining   int32
	canStart int32
	stopped  int32
	stopper  int32 // manually stop miner
	poolMode bool
	wg       sync.WaitGroup
	stopChan chan struct{}
	current  *Task
	recv     chan *types.Block

	scdo ScdoBackend
	log  *log.ScdoLog

	isFirstDownloader    int32
	isFirstBlockPrepared int32

	coinbase     common.Address
	coinbaseList []common.Address
	engine       consensus.Engine

	debtVerifier types.DebtVerifier
	msgChan      chan bool // use msgChan to receive msg setting miner to start or stop, and miner will deal with these msgs sequentially
}

// NewMiner constructs and returns a miner instance
func NewMiner(addr common.Address, addrList []common.Address, scdo ScdoBackend, verifier types.DebtVerifier, engine consensus.Engine, isPoolMode bool) *Miner {
	miner := &Miner{
		coinbase:             addr,
		coinbaseList:         addrList,   // for pool mode
		canStart:             1,          // used with downloader, canStart is 0 when downloading
		stopped:              0,          // indicate miner status (0/1), opposite to Miner.mining
		stopper:              0,          // indicate where miner could start or not. If stopper is 1, miner won't do mining
		poolMode:             isPoolMode, // whether miner is in pool mode
		scdo:                 scdo,
		wg:                   sync.WaitGroup{},
		recv:                 make(chan *types.Block, 1),
		log:                  log.GetLogger("miner"),
		isFirstDownloader:    1,
		isFirstBlockPrepared: 0,
		debtVerifier:         verifier,
		engine:               engine,
		msgChan:              make(chan bool, 100),
	}

	event.BlockDownloaderEventManager.AddListener(miner.downloaderEventCallback)
	event.TransactionInsertedEventManager.AddAsyncListener(miner.newTxOrDebtCallback)
	event.DebtsInsertedEventManager.AddAsyncListener(miner.newTxOrDebtCallback)
	go miner.handleMsg()
	return miner
}

// GetEngine gets the miner engine
func (miner *Miner) GetEngine() consensus.Engine {
	return miner.engine
}

// SetThreads sets the number of mining threads.
func (miner *Miner) SetThreads(threads int) {
	if miner.engine != nil {
		miner.engine.SetThreads(threads)
	}
}

// SetCoinbase set the coinbase.
func (miner *Miner) SetCoinbase(coinbase common.Address) {
	miner.coinbase = coinbase
}

// GetCoinbase gets the coinbase
func (miner *Miner) GetCoinbase() common.Address {
	return miner.coinbase
}

// SetStopper. If stopper is 1, miner won't do mining
func (miner *Miner) SetStopper(stopper int32) {
	miner.stopper = stopper
}

// CanStart is true when the miner is stopped and stopper == 0 and
// canStart == 1
func (miner *Miner) CanStart() bool {
	if atomic.LoadInt32(&miner.stopper) == 0 &&
		atomic.LoadInt32(&miner.stopped) == 1 &&
		atomic.LoadInt32(&miner.mining) == 0 &&
		atomic.LoadInt32(&miner.canStart) == 1 {
		return true
	} else {
		return false
	}
}

// handleMsg handles messages to start or stop the miner
func (miner *Miner) handleMsg() {
	for {
		select {
		case msg := <-miner.msgChan:
			if msg == true {
				if miner.CanStart() {
					err := miner.Start()
					if err != nil {
						miner.log.Error("error start miner,%s", err.Error())
					}
				} else {
					miner.log.Warn("cannot start miner,stopper:%d, stopped:%d,mining:%d,canStart:%d",
						atomic.LoadInt32(&miner.stopper),
						atomic.LoadInt32(&miner.stopped),
						atomic.LoadInt32(&miner.mining),
						atomic.LoadInt32(&miner.canStart))
				}
			} else {
				if atomic.LoadInt32(&miner.stopped) == 0 && atomic.LoadInt32(&miner.mining) == 1 {
					miner.Stop()

				} else {
					miner.log.Warn("miner is not working,stopper:%d, stopped:%d,mining:%d,canStart:%d",
						atomic.LoadInt32(&miner.stopper),
						atomic.LoadInt32(&miner.stopped),
						atomic.LoadInt32(&miner.mining),
						atomic.LoadInt32(&miner.canStart))
				}
			}
		}
	}
}

// Start is used to start the miner
func (miner *Miner) Start() error {
	miner.stopChan = make(chan struct{})

	if istanbul, ok := miner.engine.(consensus.Istanbul); ok {
		if err := istanbul.Start(miner.scdo.BlockChain(), miner.scdo.BlockChain().CurrentBlock, nil); err != nil {
			panic(fmt.Sprintf("failed to start istanbul engine: %v", err))
		}
	}

	// try to prepare the first block
	if err := miner.prepareNewBlock(miner.recv); err != nil {
		miner.log.Warn(err.Error())
		return err
	}

	go miner.waitBlock()
	//minerCount++
	atomic.StoreInt32(&miner.mining, 1)
	atomic.StoreInt32(&miner.stopped, 0)
	miner.log.Info("Miner started")

	return nil
}

// Stop is used to stop the miner
func (miner *Miner) Stop() {
	// set stopped to 1 to prevent restart
	atomic.StoreInt32(&miner.stopped, 1)
	miner.stopMining()
	atomic.StoreInt32(&miner.mining, 0)
	if istanbul, ok := miner.engine.(consensus.Istanbul); ok {
		if err := istanbul.Stop(); err != nil {
			panic(fmt.Sprintf("failed to stop istanbul engine: %v", err))
		}

	}

}

func (miner *Miner) stopMining() {
	// notify all threads to terminate
	if miner.stopChan != nil {
		close(miner.stopChan)
	}
	atomic.StoreInt32(&miner.mining, 0)

	// wait for all threads to terminate
	miner.wg.Wait()
	miner.log.Info("Miner stopped.")
}

// IsMining returns true if the miner is started, otherwise false
func (miner *Miner) IsMining() bool {
	return atomic.LoadInt32(&miner.mining) == 1
}

// downloaderEventCallback handles events which indicate the downloader state
func (miner *Miner) downloaderEventCallback(e event.Event) {

	switch e.(int) {
	case event.DownloaderStartEvent:
		miner.log.Info("got download start event, stop miner")
		atomic.StoreInt32(&miner.canStart, 0)
		miner.msgChan <- false

	case event.DownloaderDoneEvent, event.DownloaderFailedEvent:
		atomic.StoreInt32(&miner.canStart, 1)
		atomic.StoreInt32(&miner.isFirstDownloader, 0)
		miner.msgChan <- true
	}
}

// newTxOrDebtCallback handles the new tx event
func (miner *Miner) newTxOrDebtCallback(e event.Event) {
	miner.msgChan <- true
}

// waitBlock waits for blocks to be mined continuously
func (miner *Miner) waitBlock() {

out:
	for {
		select {
		case result := <-miner.recv:
			for {
				if result == nil {
					break
				}

				miner.log.Info("found a new mined block, height:%d, hash:%s, time:%d", result.Header.Height, result.HeaderHash.Hex(), time.Now().Unix())
				ret := miner.saveBlock(result)
				if ret != nil {
					miner.log.Error("failed to save the block, for %s", ret.Error())
					break
				}
				//mining lock

				if h, ok := miner.engine.(consensus.Handler); ok {
					h.NewChainHead()
				}

				miner.log.Info("saved mined block successfully")
				event.BlockMinedEventManager.Fire(result) // notify p2p to broadcast the block
				break
			}
			atomic.StoreInt32(&miner.stopped, 1)
			atomic.StoreInt32(&miner.mining, 0)
			// loop mining after mining completed
			miner.newTxOrDebtCallback(event.EmptyEvent)
		case <-miner.stopChan:
			break out
		}
	}
}

// newHeaderByParent creates a new header given the parent block
func newHeaderByParent(parent *types.Block, coinbase common.Address, timestamp int64) *types.BlockHeader {
	return &types.BlockHeader{
		PreviousBlockHash: parent.HeaderHash,
		Creator:           coinbase,
		Height:            parent.Header.Height + 1,
		CreateTimestamp:   big.NewInt(timestamp),
	}
}

// prepareNewBlock prepares a new block to be mined
func (miner *Miner) prepareNewBlock(recv chan *types.Block) error {
	miner.log.Debug("starting mining the new block")

	timestamp := time.Now().Unix()
	parent, stateDB, err := miner.scdo.BlockChain().GetCurrentInfo()
	if err != nil {
		return fmt.Errorf("failed to get current info, %s", err)
	}

	if parent.Header.CreateTimestamp.Cmp(new(big.Int).SetInt64(timestamp)) >= 0 {
		timestamp = parent.Header.CreateTimestamp.Int64() + 1
	}

	// this will ensure we're not going off too far in the future
	if now := time.Now().Unix(); timestamp > now+1 {
		wait := time.Duration(timestamp-now) * time.Second
		miner.log.Info("Mining too far in the future, waiting for %s", wait)
		time.Sleep(wait)
	}

	header := newHeaderByParent(parent, miner.coinbase, timestamp)
	miner.log.Debug("mining a block with coinbase %s", miner.coinbase.Hex())

	err = miner.engine.Prepare(miner.scdo.BlockChain(), header)
	if err != nil {
		return fmt.Errorf("failed to prepare header, %s", err)
	}

	if miner.poolMode {
		// pool mining mode
		miner.chooseCoinBase()
		header.Creator = miner.coinbase
	}

	if common.IsShardEnabled() {
		if coinbaseShardNum := miner.coinbase.Shard(); coinbaseShardNum != common.LocalShardNumber {
			return fmt.Errorf("invalid coinbase, shard number is [%v], but local shard number is [%v]", coinbaseShardNum, common.LocalShardNumber)
		}
	}

	miner.current = NewTask(header, miner.coinbase, miner.debtVerifier)
	err = miner.current.applyTransactionsAndDebts(miner.scdo, stateDB, miner.scdo.BlockChain().AccountDB(), miner.log)
	if err != nil {
		return fmt.Errorf("failed to apply transaction %s", err)
	}

	if miner.poolMode {
		miner.log.Info("create a new task for the pool, height:%d, difficult:%d", header.Height, header.Difficulty)
		preBlock := miner.current.generateBlock()
		miner.current.header = preBlock.Header.Clone()
	} else {
		miner.log.Info("committing a new task to engine, height:%d, difficult:%d", header.Height, header.Difficulty)
		miner.commitTask(miner.current, recv)
	}
	return nil
}

// saveBlock saves the block in the given result to the blockchain
func (miner *Miner) saveBlock(result *types.Block) error {
	now := time.Now()
	// entrance
	memory.Print(miner.log, "miner saveBlock entrance", now, false)
	txPool := miner.scdo.TxPool().Pool

	ret := miner.scdo.BlockChain().WriteBlock(result, txPool)

	// entrance
	memory.Print(miner.log, "miner saveBlock exit", now, true)

	return ret
}

// commitTask commits the given task to the miner
func (miner *Miner) commitTask(task *Task, recv chan *types.Block) {
	block := task.generateBlock()
	miner.engine.Seal(miner.scdo.BlockChain(), block, miner.stopChan, recv)
}

//GetWork get the current task in a printable format
func (miner *Miner) GetWork() map[string]interface{} {
	if miner.current == nil {
		miner.log.Info("there is no task so far")
		return nil
	}
	task := miner.current
	return PrintableOutputTask(task)
}

// GetWorkTask gets the current task
func (miner *Miner) GetWorkTask() *Task {
	return miner.current
}

// GetCurrentWorkHeader returns the header of current task
func (miner *Miner) GetCurrentWorkHeader(totalDifficulty *big.Int) map[string]interface{} {
	task := miner.GetWorkTask()
	if task == nil {
		miner.log.Info("there is no task so far")
		return nil
	}
	newTotalDifficulty := big.NewInt(0)
	newTotalDifficulty.Add(totalDifficulty, task.header.Difficulty)
	return PrintableOutputTaskHeader(task.header, newTotalDifficulty)
}

// SubmitWork is used to submit the nonce to generate the final block
func (miner *Miner) SubmitWork(height uint64, nonce uint64) error {

	// validate nonce based on miner.current
	// If valid, create a block and pass it into miner.recv
	if miner.current == nil {
		return errors.New("there is no task so far")
	}

	if miner.current.header.Height != height {
		return errors.New("Height not match")
	}

	taskHeader := miner.current.header.Clone()
	taskHeader.Witness = []byte(strconv.FormatUint(nonce, 10))

	err := miner.engine.VerifyHeader(miner.scdo.BlockChain(), taskHeader)
	if err != nil {
		return err
	}
	miner.current.header.Witness = taskHeader.Witness
	block := miner.current.generateBlock()
	miner.current = nil
	miner.recv <- block
	return nil

}

// GetTaskDifficulty gets the difficulty of current task
func (miner *Miner) GetTaskDifficulty() *big.Int {

	if miner.current == nil {
		miner.log.Info("there is no task so far")
		return nil
	}
	difficulty := miner.current.header.Difficulty
	if difficulty == nil {
		return nil
	}
	return difficulty
}

// chooseCoinBase selects the coinbase randomly from the given list
func (miner *Miner) chooseCoinBase() {
	if len(miner.coinbaseList) == 0 {
		return
	}
	index := rand.Intn(len(miner.coinbaseList))
	miner.coinbase = miner.coinbaseList[index]
}
