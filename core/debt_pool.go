/**
* @file
* @copyright defined in scdo/LICENSE
 */

package core

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/core/state"
	"github.com/scdoproject/go-scdo/core/types"
	"github.com/scdoproject/go-scdo/event"
	"github.com/scdoproject/go-scdo/log"
)

const debtTimeoutDuration = 3 * time.Hour

// DebtPool debt pool
type DebtPool struct {
	*Pool
	verifier         types.DebtVerifier
	toConfirmedDebts *ConcurrentDebtMap
}

// NewDebtPool creates and returns a new debt pool
func NewDebtPool(chain blockchain, verifier types.DebtVerifier) *DebtPool {
	log := log.GetLogger("debtpool")

	getObjectFromBlock := func(block *types.Block) []poolObject {
		return debtsToObjects(block.Debts)
	}
	// 1st bool: can remove from object pool
	// 2nd bool: can remove from cachedTxs
	canRemove := func(chain blockchain, state *state.Statedb, item *poolItem) (bool, bool) {
		nowTimestamp := time.Now()
		duration := nowTimestamp.Sub(item.timestamp)
		if duration > debtTimeoutDuration {
			log.Debug("remove debt %s because not packed for more than three hours", item.GetHash().Hex())
			return true, true
		}
		debtIndex, err := chain.GetStore().GetDebtIndex(item.GetHash())
		if err != nil || debtIndex == nil {
			return false, false
		}

		return true, false
	}

	objectValidation := func(state *state.Statedb, obj poolObject) error {
		// skip as we already check before adding
		return nil
	}

	afterAdd := func(obj poolObject) {
		log.Debug("receive debt and add it. debt hash: %v, time: %d", obj.GetHash(), time.Now().UnixNano())

		event.DebtsInsertedEventManager.Fire(obj.(*types.Debt))
	}
	cachedTxs := NewCachedTxs(0)
	pool := NewPool(DebtPoolCapacity, chain, getObjectFromBlock, canRemove, log, objectValidation, afterAdd, cachedTxs)

	debtPool := &DebtPool{
		Pool:             pool,
		verifier:         verifier,
		toConfirmedDebts: NewConcurrentDebtMap(ToConfirmedDebtCapacity),
	}

	go debtPool.loopCheckingDebt()

	return debtPool
}

// loopCheckingDebt check whether debt is confirmed.
// we only add debt to pool when it is confirmed
func (dp *DebtPool) loopCheckingDebt() {
	if dp.verifier == nil {
		dp.log.Info("exit checking as verifier is nil")
		return
	}

	for {
		if dp.toConfirmedDebts.count() == 0 {
			time.Sleep(10 * time.Second)
		} else {
			err := dp.DoMulCheckingDebt()
			if err != nil {
				dp.log.Warn("multiple threads checking error: %s", err)
				// need to sleep some time
				time.Sleep(5 * time.Second)
			}
		}
	}
}

// DoMulCheckingDebt use multiple threads to validate debts
func (dp *DebtPool) DoMulCheckingDebt() error {
	tmp := dp.toConfirmedDebts.getList()
	len := len(tmp)
	threads := runtime.NumCPU() / 2
	dp.log.Info("use %d threads to validate debts", threads)
	// single thread for few CPU kernel or few txs to validate.
	if threads <= 1 || len < threads {
		for i := 0; i < len; i++ {
			if err := dp.DoMulCheckingDebtHandler(tmp[i]); err != nil {
				return err
			}
		}
		return nil
	}
	// parallel validates txs
	var err error
	var hasErr uint32
	wg := sync.WaitGroup{}

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()
			for j := offset; j < len && atomic.LoadUint32(&hasErr) == 0; j += threads {
				if e := dp.DoMulCheckingDebtHandler(tmp[j]); e != nil {
					if atomic.CompareAndSwapUint32(&hasErr, 0, 1) {
						err = e
					}
					break
				}
			}
		}(i)
	}
	wg.Wait()
	return err
}

// DoMulCheckingDebtHandler DoMulCheckingDebt handler
func (dp *DebtPool) DoMulCheckingDebtHandler(d *types.Debt) error {
	recoverable, err := d.Validate(dp.verifier, false, common.LocalShardNumber)
	if err != nil {
		if recoverable {
			dp.log.Debug("check debt with recoverable error %s", err)
		} else {
			dp.log.Info("check debt with unrecoverable error %s", err)
			dp.toConfirmedDebts.removeByValue(d)
		}
		return err
	} else {
		// confirmed
		err := dp.addToPool(d)
		if err == nil {
			// remove if success
			dp.toConfirmedDebts.removeByValue(d)
			return nil
		} else {
			return err
		}
	}
}

// DoCheckingDebt is a legecy rountine
func (dp *DebtPool) DoCheckingDebt() {
	tmp := dp.toConfirmedDebts.items()
	for h, d := range tmp {
		recoverable, err := d.Validate(dp.verifier, false, common.LocalShardNumber)
		if err != nil {
			if recoverable {
				dp.log.Debug("check debt with recoverable error %s", err)
			} else {
				dp.log.Info("check debt with unrecoverable error %s", err)
				dp.toConfirmedDebts.remove(h)
			}
		} else {
			// confirmed
			err := dp.addToPool(d)
			if err == nil {
				// remove if success
				dp.toConfirmedDebts.remove(h)
			}
		}
	}
}

// AddDebtArray adds an array of debts to the debt pool
func (dp *DebtPool) AddDebtArray(debts []*types.Debt) {
	for _, d := range debts {
		dp.AddDebt(d)
	}

	dp.log.Debug("add %d debts, cap %d", len(debts), dp.getObjectCount(true, true))
}

// AddDebt adds a debt to toConfirmedDebts
func (dp *DebtPool) AddDebt(debt *types.Debt) error {
	if debt == nil {
		return nil
	}

	// skip if already exist
	if dp.toConfirmedDebts.has(debt.Hash) {
		return nil
	}

	if dp.GetObject(debt.Hash) != nil {
		return nil
	}

	err := dp.toConfirmedDebts.add(debt)
	if err != nil {
		dp.log.Warn("add debts to to be confirmed pool failed debt hash:%s, err: %s.", debt.Hash, err)
	}

	return err
}

// addToPool adds a debt to the debt pool
func (dp *DebtPool) addToPool(debt *types.Debt) error {
	err := dp.addObject(debt)
	if err != nil {
		dp.log.Warn("add debts failed debt hash:%s, err: %s.", debt.Hash, err)
	}

	return err
}

// GetProcessableDebts gets processable debts given the total size from the debt pool
func (dp *DebtPool) GetProcessableDebts(size int) ([]*types.Debt, int) {
	objects, remainSize := dp.getProcessableObjects(size)

	return objectsToDebts(objects), remainSize
}

// objectsToDebts converts objects to debts
func objectsToDebts(objects []poolObject) []*types.Debt {
	results := make([]*types.Debt, len(objects))
	for index, obj := range objects {
		results[index] = obj.(*types.Debt)
	}

	return results
}

// objectsToDebts converts debts to objects
func debtsToObjects(debts []*types.Debt) []poolObject {
	objects := make([]poolObject, len(debts))

	for index, d := range debts {
		objects[index] = d
	}

	return objects
}

// GetDebtByHash gets debt from the debt pool by hash
func (dp *DebtPool) GetDebtByHash(hash common.Hash) *types.Debt {
	debt := dp.toConfirmedDebts.get(hash)
	if debt != nil {
		return debt
	}

	obj := dp.GetObject(hash)
	if obj != nil {
		return obj.(*types.Debt)
	}

	return nil
}

// RemoveDebtByHash removes debt from the debt pool by hash
func (dp *DebtPool) RemoveDebtByHash(hash common.Hash) {
	dp.toConfirmedDebts.remove(hash)
	dp.removeOject(hash)
}

// GetDebts get debts based on their status
func (dp *DebtPool) GetDebts(processing, pending bool) []*types.Debt {
	objects := dp.getObjects(processing, pending)
	debts := objectsToDebts(objects)

	if pending {
		debts = append(debts, dp.toConfirmedDebts.getList()...)
	}

	return debts
}

// GetDebtCount returns the count of debts in the pool
func (dp *DebtPool) GetDebtCount(processing, pending bool) int {
	count := dp.getObjectCount(processing, pending)
	if pending {
		count += dp.toConfirmedDebts.count()
	}

	return count
}
