/**
* @file
* @copyright defined in scdo/LICENSE
 */

package core

import (
	"container/heap"
	"sort"

	"github.com/scdoproject/go-scdo/common"
)

// txCollection represents the nonce sorted transactions of an account.
type txCollection struct {
	txs       map[uint64]*poolItem
	nonceHeap *common.Heap
}

// newTxCollection creates a new txCollection
func newTxCollection() *txCollection {
	return &txCollection{
		txs: make(map[uint64]*poolItem),
		nonceHeap: common.NewHeap(func(i, j common.HeapItem) bool {
			iNonce := i.(*poolItem).Nonce()
			jNonce := j.(*poolItem).Nonce()
			return iNonce < jNonce
		}),
	}
}

// add adds a tx to the txCollection
func (collection *txCollection) add(tx *poolItem) bool {
	if existTx := collection.txs[tx.Nonce()]; existTx != nil {
		existTx.poolObject = tx.poolObject
		existTx.timestamp = tx.timestamp
		return false
	}

	heap.Push(collection.nonceHeap, tx)
	collection.txs[tx.Nonce()] = tx

	return true
}

// get returns the tx given nonce
func (collection *txCollection) get(nonce uint64) *poolItem {
	return collection.txs[nonce]
}

// remove removes the tx from the txCollection given nonce
func (collection *txCollection) remove(nonce uint64) bool {
	if tx := collection.txs[nonce]; tx != nil {
		heap.Remove(collection.nonceHeap, tx.GetHeapIndex())
		delete(collection.txs, nonce)
		return true
	}

	return false
}

// len returns the length of the nonceHeap of the txCollection
func (collection *txCollection) len() int {
	return collection.nonceHeap.Len()
}

// peek returns the top item of the nonceHeap of the txCollection
func (collection *txCollection) peek() *poolItem {
	if item := collection.nonceHeap.Peek(); item != nil {
		return item.(*poolItem)
	}

	return nil
}

// pop pops up the top item of the nonceHeap of the txCollection
func (collection *txCollection) pop() *poolItem {
	tx := heap.Pop(collection.nonceHeap).(*poolItem)
	delete(collection.txs, tx.Nonce())
	return tx
}

// list lists all the txs in the txCollection
func (collection *txCollection) list() []poolObject {
	result := make([]poolObject, len(collection.txs))
	i := 0

	for _, tx := range collection.txs {
		result[i] = tx.poolObject
		i++
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Nonce() < result[j].Nonce()
	})

	return result
}

// cmp compares to the specified tx collection based on price and timestamp.
//   For higher price, return 1.
//   For lower price, return -1.
//   Otherwise:
//     For earier timestamp, return 1.
//     For later timestamp, return -1.
//     Otherwise, return 0.
func (collection *txCollection) cmp(other *txCollection) int {
	if other == nil {
		return 1
	}

	iTx, jTx := collection.peek(), other.peek()
	if iTx == nil && jTx == nil {
		return 0
	}

	if jTx == nil {
		return 1
	}

	if iTx == nil {
		return -1
	}

	if r := iTx.Price().Cmp(jTx.Price()); r != 0 {
		return r
	}

	if iTx.timestamp.Before(jTx.timestamp) {
		return 1
	}

	if iTx.timestamp.After(jTx.timestamp) {
		return -1
	}

	return 0
}
