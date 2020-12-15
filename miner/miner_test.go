/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package miner

import (
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/consensus/factory"
	"github.com/scdoproject/go-scdo/core"
	"github.com/scdoproject/go-scdo/core/types"
	"github.com/scdoproject/go-scdo/crypto"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var defaultMinerAddr = common.BytesToAddress([]byte{1})
var scdo = NewTestScdoBackend()

func Test_NewMiner(t *testing.T) {
	miner := createMiner()

	assert.Equal(t, miner != nil, true)
	checkMinerMembers(miner, defaultMinerAddr, scdo, t)

	assert.Equal(t, miner.GetCoinbase(), defaultMinerAddr)
	assert.Equal(t, miner.IsMining(), false)
}

func Test_SetCoinbase(t *testing.T) {
	miner := createMiner()

	assert.Equal(t, miner.GetCoinbase(), defaultMinerAddr)

	newAddr := common.BytesToAddress([]byte{2})
	miner.SetCoinbase(newAddr)
	assert.Equal(t, miner.GetCoinbase(), newAddr)
}

func Test_Start(t *testing.T) {
	// Init LevelDB
	dir := prepareDbFolder("", "leveldbtest")
	defer os.RemoveAll(dir)

	miner := createMiner()
	miner.mining = 1

	err := miner.Start()
	assert.Equal(t, err, ErrMinerIsRunning)

	miner.mining = 0
	miner.canStart = 0
	err = miner.Start()
	assert.Equal(t, err, ErrNodeIsSyncing)

	miner.canStart = 1
	err = miner.Start()
	assert.Equal(t, err, nil)

	assert.Equal(t, miner.stopped, int32(0))
	assert.Equal(t, miner.mining, int32(1))
	miner.Stop()
	assert.Equal(t, miner.stopped, int32(1))
	assert.Equal(t, miner.mining, int32(0))
}

func Test_MinerPack(t *testing.T) {
	verifier := types.NewTestVerifier(true, true, nil)
	minerPackWithVerifier(t, verifier)
}

func Test_MinerPackWithVerifier(t *testing.T) {
	verifier := types.NewTestVerifierWithFunc(func(debt *types.Debt) (bool, bool, error) {
		a := debt.Hash.Big().Uint64()
		if a%2 == 0 {
			return true, true, nil
		} else {
			return true, false, nil
		}
	})

	minerPackWithVerifier(t, verifier)
}

func minerPackWithVerifier(t *testing.T, verifier types.DebtVerifier) {
	backend := NewTestScdoBackendWithVerifier(verifier)
	backend.TxPool().SetLogLevel(logrus.WarnLevel)
	backend.DebtPool().SetLogLevel(logrus.WarnLevel)

	// init pool
	totalDebtCount := 10000
	confirmedDebtCount := 0
	for i := 0; i < totalDebtCount; i++ {
		d := types.NewTestDebtWithTargetShard(types.TestGenesisShard)
		_, c, _ := verifier.ValidateDebt(d)
		if c {
			confirmedDebtCount++
		}

		err := backend.DebtPool().AddDebt(d)
		assert.Nil(t, err)
	}
	backend.debtPool.DoCheckingDebt()

	totalTxCount := 10000
	for i := 0; i < totalTxCount/2; i++ {
		err := backend.TxPool().AddTransaction(types.NewTestTransactionWithNonce(uint64(i)))
		assert.Nil(t, err)
	}

	for i := 0; i < totalTxCount/2; i++ {
		err := backend.TxPool().AddTransaction(types.NewTestCrossShardTransactionWithNonce(uint64(i + 5000)))
		assert.Nil(t, err)
	}

	// init miner
	coinbase := *crypto.MustGenerateShardAddress(types.TestGenesisShard)
	miner := NewMiner(coinbase, backend, verifier, factory.MustGetConsensusEngine(common.Sha256Algorithm))
	miner.log.SetLevel(logrus.WarnLevel)
	miner.mining = 1

	debtCount := 0
	txCount := 0

	// first pack
	resultBlock := mineNewBlock(t, miner)
	debtCount += len(resultBlock.Debts)
	txCount += len(resultBlock.Transactions)
	assert.Equal(t, totalDebtCount-debtCount, backend.debtPool.GetDebtCount(true, true))
	assert.Equal(t, totalTxCount-txCount+1, backend.txPool.GetTxCount())

	// second pack
	resultBlock = mineNewBlock(t, miner)
	debtCount += len(resultBlock.Debts)
	txCount += len(resultBlock.Transactions)
	assert.Equal(t, totalDebtCount-debtCount, backend.debtPool.GetDebtCount(true, true))
	assert.Equal(t, totalTxCount-txCount+2, backend.txPool.GetTxCount())

	// third pack
	resultBlock = mineNewBlock(t, miner)
	debtCount += len(resultBlock.Debts)
	txCount += len(resultBlock.Transactions)
	assert.Equal(t, totalDebtCount-debtCount, backend.debtPool.GetDebtCount(true, true))
	assert.Equal(t, totalTxCount-txCount+3, backend.txPool.GetTxCount())

	assert.Equal(t, confirmedDebtCount, debtCount)
	assert.Equal(t, totalTxCount+3, txCount)
}

func mineNewBlock(t *testing.T, miner *Miner) *types.Block {
	recv := make(chan *types.Block)
	err := miner.prepareNewBlock(recv)
	assert.Nil(t, err)

	wg := sync.WaitGroup{}
	wg.Add(1)

	var resultBlock *types.Block
	go func() {
		defer wg.Done()
		resultBlock = <-recv
	}()

	wg.Wait()

	bc := miner.scdo.BlockChain()
	err = bc.WriteBlock(resultBlock)
	assert.Nil(t, err)
	oldHeader := bc.GetHeaderByHeight(resultBlock.Header.Height - 1).Hash()
	miner.scdo.TxPool().HandleChainHeaderChanged(resultBlock.HeaderHash, oldHeader)
	miner.scdo.DebtPool().HandleChainHeaderChanged(resultBlock.HeaderHash, oldHeader)

	return resultBlock
}

func createMiner() *Miner {
	return NewMiner(defaultMinerAddr, scdo, nil, factory.MustGetConsensusEngine(common.Sha256Algorithm))
}

func checkMinerMembers(miner *Miner, addr common.Address, scdo ScdoBackend, t *testing.T) {
	assert.Equal(t, miner.coinbase, addr)

	assert.Equal(t, miner.mining, int32(0))
	assert.Equal(t, miner.canStart, int32(1))
	assert.Equal(t, miner.stopped, int32(0))
	assert.Equal(t, miner.scdo, scdo)
	assert.Equal(t, miner.isFirstDownloader, int32(1))
	assert.Equal(t, miner.isFirstBlockPrepared, int32(0))
	assert.Equal(t, miner.isFirstDownloader, int32(1))
}

// TestScdoBackend implements the ScdoBackend interface.
type TestScdoBackend struct {
	txPool     *core.TransactionPool
	debtPool   *core.DebtPool
	blockchain *core.Blockchain
}

func NewTestScdoBackend() *TestScdoBackend {
	return NewTestScdoBackendWithVerifier(nil)
}

func NewTestScdoBackendWithVerifier(verifier types.DebtVerifier) *TestScdoBackend {
	scdoBackend := &TestScdoBackend{}

	scdoBackend.blockchain = core.NewTestBlockchainWithVerifier(verifier)
	scdoBackend.debtPool = core.NewDebtPool(scdoBackend.blockchain, verifier)
	scdoBackend.txPool = core.NewTransactionPool(*core.DefaultTxPoolConfig(), scdoBackend.blockchain)

	return scdoBackend
}

func (t TestScdoBackend) TxPool() *core.TransactionPool {
	return t.txPool
}

func (t TestScdoBackend) DebtPool() *core.DebtPool {
	return t.debtPool
}

func (t TestScdoBackend) BlockChain() *core.Blockchain {
	return t.blockchain
}

func prepareDbFolder(pathRoot string, subDir string) string {
	dir, err := ioutil.TempDir(pathRoot, subDir)
	if err != nil {
		panic(err)
	}

	return dir
}
