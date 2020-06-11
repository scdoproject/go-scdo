/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package miner

import (
	"math/big"
	"time"

	"github.com/seelecredo/go-seelecredo/common"
	"github.com/seelecredo/go-seelecredo/common/memory"
	"github.com/seelecredo/go-seelecredo/consensus"
	"github.com/seelecredo/go-seelecredo/core"
	"github.com/seelecredo/go-seelecredo/core/state"
	"github.com/seelecredo/go-seelecredo/core/txs"
	"github.com/seelecredo/go-seelecredo/core/types"
	"github.com/seelecredo/go-seelecredo/database"
	"github.com/seelecredo/go-seelecredo/log"
)

// Task is a mining work for engine, containing block header, transactions, and transaction receipts.
type Task struct {
	header   *types.BlockHeader
	txs      []*types.Transaction
	receipts []*types.Receipt
	debts    []*types.Debt

	coinbase     common.Address
	debtVerifier types.DebtVerifier
}

// NewTask return Task object
func NewTask(header *types.BlockHeader, coinbase common.Address, verifier types.DebtVerifier) *Task {
	return &Task{
		header:       header,
		coinbase:     coinbase,
		debtVerifier: verifier,
	}
}

// applyTransactionsAndDebts TODO need to check more about the transactions, such as gas limit
func (task *Task) applyTransactionsAndDebts(seeleCredo SlcBackend, statedb *state.Statedb, accountStateDB database.Database, log *log.SeeleCredoLog) error {
	now := time.Now()
	// entrance
	memory.Print(log, "task applyTransactionsAndDebts entrance", now, false)

	// choose transactions from the given txs
	size := task.chooseDebts(seeleCredo, statedb, log)

	// the reward tx will always be at the first of the block's transactions
	reward, err := task.handleMinerRewardTx(statedb)
	if err != nil {
		return err
	}

	task.chooseTransactions(seeleCredo, statedb, log, size)

	log.Info("mining block height:%d, reward:%s, transaction number:%d, debt number: %d",
		task.header.Height, reward, len(task.txs), len(task.debts))

	batch := accountStateDB.NewBatch()
	root, err := statedb.Commit(batch)
	if err != nil {
		return err
	}

	task.header.StateHash = root

	// exit
	memory.Print(log, "task applyTransactionsAndDebts exit", now, true)

	return nil
}

func (task *Task) chooseDebts(seeleCredo SlcBackend, statedb *state.Statedb, log *log.SeeleCredoLog) int {
	now := time.Now()
	// entrance
	memory.Print(log, "task chooseDebts entrance", now, false)

	size := core.BlockByteLimit

	for size > 0 {
		debts, _ := seeleCredo.DebtPool().GetProcessableDebts(size)
		if len(debts) == 0 {
			break
		}

		for _, d := range debts {
			err := seeleCredo.BlockChain().ApplyDebtWithoutVerify(statedb, d, task.coinbase)
			if err != nil {
				log.Debug("apply debt error %s", err)
				seeleCredo.DebtPool().RemoveDebtByHash(d.Hash)
				continue
			}

			size = size - d.Size()
			task.debts = append(task.debts, d)
		}
	}

	// exit
	memory.Print(log, "task chooseDebts exit", now, true)

	return size
}

// handleMinerRewardTx handles the miner reward transaction.
func (task *Task) handleMinerRewardTx(statedb *state.Statedb) (*big.Int, error) {
	reward := consensus.GetReward(task.header.Height)
	rewardTx, err := txs.NewRewardTx(task.coinbase, reward, task.header.CreateTimestamp.Uint64())
	if err != nil {
		return nil, err
	}

	rewardTxReceipt, err := txs.ApplyRewardTx(rewardTx, statedb)
	if err != nil {
		return nil, err
	}

	task.txs = append(task.txs, rewardTx)

	// add the receipt of the reward tx
	task.receipts = append(task.receipts, rewardTxReceipt)

	return reward, nil
}

func (task *Task) chooseTransactions(seeleCredo SlcBackend, statedb *state.Statedb, log *log.SeeleCredoLog, size int) {
	now := time.Now()
	// entrance
	memory.Print(log, "task chooseTransactions entrance", now, false)

	txIndex := 1 // the first tx is miner reward

	for size > 0 {
		txs, txsSize := seeleCredo.TxPool().GetProcessableTransactions(size)
		if len(txs) == 0 {
			break
		}

		for _, tx := range txs {
			if err := tx.Validate(statedb, task.header.Height); err != nil {
				seeleCredo.TxPool().RemoveTransaction(tx.Hash)
				log.Error("failed to validate tx %s, for %s", tx.Hash.Hex(), err)
				txsSize = txsSize - tx.Size()
				continue
			}

			receipt, err := seeleCredo.BlockChain().ApplyTransaction(tx, txIndex, task.coinbase, statedb, task.header)
			if err != nil {
				seeleCredo.TxPool().RemoveTransaction(tx.Hash)
				log.Error("failed to apply tx %s, %s", tx.Hash.Hex(), err)
				txsSize = txsSize - tx.Size()
				continue
			}

			task.txs = append(task.txs, tx)
			task.receipts = append(task.receipts, receipt)
			txIndex++
		}

		size -= txsSize
	}

	// exit
	memory.Print(log, "task chooseTransactions exit", now, true)
}

// generateBlock builds a block from task
func (task *Task) generateBlock() *types.Block {
	return types.NewBlock(task.header, task.txs, task.receipts, task.debts)
}

// Result is the result mined by engine. It contains the raw task and mined block.
type Result struct {
	task  *Task
	block *types.Block // mined block, with good nonce
}

func PrintableOutputTask(task *Task) map[string]interface{} {

	result := map[string]interface{}{
		"header":   task.header,
		"txs":      task.txs,
		"receipts": task.receipts,
		"debts":    task.debts,
		"coinbase": task.coinbase,
	}
	return result
}
