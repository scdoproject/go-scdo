/**
* @file
* @copyright defined in scdo/LICENSE
 */

package store

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/core/types"
	"github.com/scdoproject/go-scdo/database"
	"github.com/syndtr/goleveldb/leveldb/errors"
)

var (
	keyHeadBlockHash = []byte("HeadBlockHash")

	keyPrefixHash          = []byte("H")
	keyPrefixHeader        = []byte("h")
	keyPrefixTD            = []byte("t")
	keyPrefixBody          = []byte("b")
	keyPrefixReceipts      = []byte("r")
	keyPrefixDirtyAccounts = []byte("D")
	keyPrefixTxIndex       = []byte("i")
	keyPrefixDebtIndex     = []byte("d")
)

// blockBody represents the payload of a block
type blockBody struct {
	Txs   []*types.Transaction // Txs is a transaction collection
	Debts []*types.Debt        // Debts is a debt collection
}

// blockchainDatabase wraps a database used for the blockchain
type blockchainDatabase struct {
	db database.Database
}

// NewBlockchainDatabase returns a blockchainDatabase instance.
// There are following mappings in database:
//   1) keyPrefixHash + height => hash
//   2) keyHeadBlockHash => HEAD hash
//   3) keyPrefixHeader + hash => header
//   4) keyPrefixTD + hash => total difficulty (td for short)
//   5) keyPrefixBody + hash => block body (transactions)
//   6) keyPrefixReceipts + hash => block receipts
//   7) keyPrefixTxIndex + txHash => txIndex
func NewBlockchainDatabase(db database.Database) BlockchainStore {
	return &blockchainDatabase{db}
}

func heightToHashKey(height uint64) []byte      { return append(keyPrefixHash, encodeBlockHeight(height)...) }
func hashToHeaderKey(hash []byte) []byte        { return append(keyPrefixHeader, hash...) }
func hashToTDKey(hash []byte) []byte            { return append(keyPrefixTD, hash...) }
func hashToBodyKey(hash []byte) []byte          { return append(keyPrefixBody, hash...) }
func hashToReceiptsKey(hash []byte) []byte      { return append(keyPrefixReceipts, hash...) }
func hashToDirtyAccountsKey(hash []byte) []byte { return append(keyPrefixDirtyAccounts, hash...) }
func txHashToIndexKey(txHash []byte) []byte     { return append(keyPrefixTxIndex, txHash...) }
func debtHashToIndexKey(debtHash []byte) []byte { return append(keyPrefixDebtIndex, debtHash...) }

// GetBlockHash gets the hash of the block with the specified height in the blockchain database
func (store *blockchainDatabase) GetBlockHash(height uint64) (common.Hash, error) {
	hashBytes, err := store.db.Get(heightToHashKey(height))
	if err != nil {
		return common.EmptyHash, err
	}

	return common.BytesToHash(hashBytes), nil
}

// PutBlockHash puts the given block height which is encoded as the key
// and hash as the value to the blockchain database.
func (store *blockchainDatabase) PutBlockHash(height uint64, hash common.Hash) error {
	return store.db.Put(heightToHashKey(height), hash.Bytes())
}

// DeleteBlockHash deletes the block hash mapped to by the specified height from the blockchain database
func (store *blockchainDatabase) DeleteBlockHash(height uint64) (bool, error) {
	key := heightToHashKey(height)

	found, err := store.db.Has(key)
	if err != nil {
		return false, err
	}

	if !found {
		return false, nil
	}

	if err = store.db.Delete(key); err != nil {
		return false, err
	}

	return true, nil
}

// encodeBlockHeight encodes a block height as big endian uint64
func encodeBlockHeight(height uint64) []byte {
	encoded := make([]byte, 8)
	binary.BigEndian.PutUint64(encoded, height)
	return encoded
}

// GetHeadBlockHash gets the HEAD block hash in the blockchain database
func (store *blockchainDatabase) GetHeadBlockHash() (common.Hash, error) {
	hashBytes, err := store.db.Get(keyHeadBlockHash)
	if err != nil {
		return common.EmptyHash, err
	}

	return common.BytesToHash(hashBytes), nil
}

// PutHeadBlockHash writes the HEAD block hash into the store.
func (store *blockchainDatabase) PutHeadBlockHash(hash common.Hash) error {
	return store.db.Put(keyHeadBlockHash, hash.Bytes())
}

// GetBlockHeader gets the header of the block with the specified hash in the blockchain database
func (store *blockchainDatabase) GetBlockHeader(hash common.Hash) (*types.BlockHeader, error) {
	headerBytes, err := store.db.Get(hashToHeaderKey(hash.Bytes()))
	if err != nil {
		return nil, err
	}

	header := new(types.BlockHeader)
	if err := common.Deserialize(headerBytes, header); err != nil {
		return nil, err
	}

	return header, nil
}

// HasBlock indicates if the block with the specified hash exists in the blockchain database
func (store *blockchainDatabase) HasBlock(hash common.Hash) (bool, error) {
	key := hashToHeaderKey(hash.Bytes())

	found, err := store.db.Has(key)
	if err != nil {
		return false, err
	}

	return found, nil
}

// PutBlockHeader serializes the given block header of the block with the specified hash
// and total difficulty into the blockchain database.
// isHead indicates if the given header is the HEAD block header
func (store *blockchainDatabase) PutBlockHeader(hash common.Hash, header *types.BlockHeader, td *big.Int, isHead bool) error {
	return store.putBlockInternal(hash, header, nil, td, isHead)
}

func (store *blockchainDatabase) putBlockInternal(hash common.Hash, header *types.BlockHeader, body *blockBody, td *big.Int, isHead bool) error {
	if header == nil {
		panic("header is nil")
	}

	headerBytes := common.SerializePanic(header)

	hashBytes := hash.Bytes()

	batch := store.db.NewBatch()
	batch.Put(hashToHeaderKey(hashBytes), headerBytes)
	batch.Put(hashToTDKey(hashBytes), common.SerializePanic(td))

	if body != nil {
		batch.Put(hashToBodyKey(hashBytes), common.SerializePanic(body))
	}

	if isHead {
		// delete old txs/debts indices in old canonical chain if exists
		oldHash, err := store.GetBlockHash(header.Height)
		if err != nil && err != errors.ErrNotFound {
			return err
		}

		if err == nil {
			oldBlock, err := store.GetBlock(oldHash)

			if err != nil && err != errors.ErrNotFound {
				return err
			}

			if err == nil {
				store.batchDeleteIndices(batch, oldHash, oldBlock.Transactions, oldBlock.Debts)
			}
		}

		// add or update txs/debts indices of new HEAD block
		if body != nil {
			store.batchAddIndices(batch, hash, body.Txs, body.Debts)
		}

		// update height to hash map in canonical chain and HEAD block hash
		batch.Put(heightToHashKey(header.Height), hashBytes)
		batch.Put(keyHeadBlockHash, hashBytes)
	}

	return batch.Commit()
}

// DeleteBlockHeader deletes the block header of the specified block hash.
func (store *blockchainDatabase) DeleteBlockHeader(hash common.Hash) error {
	hashBytes := hash.Bytes()
	batch := store.db.NewBatch()

	// delete header, TD and receipts if any.
	headerKey := hashToHeaderKey(hashBytes)
	tdKey := hashToTDKey(hashBytes)
	receiptsKey := hashToReceiptsKey(hashBytes)
	if err := store.delete(batch, headerKey, tdKey, receiptsKey); err != nil {
		return err
	}

	return batch.Commit()
}

// GetBlockTotalDifficulty gets the total difficulty of the block with the specified hash in the blockchain database
func (store *blockchainDatabase) GetBlockTotalDifficulty(hash common.Hash) (*big.Int, error) {
	tdBytes, err := store.db.Get(hashToTDKey(hash.Bytes()))
	if err != nil {
		return nil, err
	}

	td := new(big.Int)
	if err = common.Deserialize(tdBytes, td); err != nil {
		return nil, err
	}

	return td, nil
}

// RecoverHeightToBlockMap recovers the height-to-block map
func (store *blockchainDatabase) RecoverHeightToBlockMap(block *types.Block) error {
	batch := store.db.NewBatch()
	// add or update txs/debts indices of this block
	store.batchAddIndices(batch, block.HeaderHash, block.Transactions, block.Debts)
	// update height to hash map in the chain
	hashBytes := block.HeaderHash.Bytes()
	batch.Put(heightToHashKey(block.Header.Height), hashBytes)
	return batch.Commit()
}

// PutBlock serializes the given block with the specified total difficulty into the blockchain database.
// isHead indicates if the block is the header block
func (store *blockchainDatabase) PutBlock(block *types.Block, td *big.Int, isHead bool) error {
	if block == nil {
		panic("block is nil")
	}

	return store.putBlockInternal(block.HeaderHash, block.Header, &blockBody{block.Transactions, block.Debts}, td, isHead)
}

// GetBlock gets the block with the specified hash in the blockchain database
func (store *blockchainDatabase) GetBlock(hash common.Hash) (*types.Block, error) {
	header, err := store.GetBlockHeader(hash)
	if err != nil {
		return nil, err
	}

	bodyKey := hashToBodyKey(hash.Bytes())
	hasBody, err := store.db.Has(bodyKey)
	if err != nil {
		return nil, err
	}

	if !hasBody {
		return &types.Block{
			HeaderHash: hash,
			Header:     header,
		}, nil
	}

	bodyBytes, err := store.db.Get(bodyKey)
	if err != nil {
		return nil, err
	}

	body := blockBody{}
	if err := common.Deserialize(bodyBytes, &body); err != nil {
		return nil, err
	}

	return &types.Block{
		HeaderHash:   hash,
		Header:       header,
		Transactions: body.Txs,
		Debts:        body.Debts,
	}, nil
}

// DeleteBlock deletes the block of the specified block hash.
func (store *blockchainDatabase) DeleteBlock(hash common.Hash) error {
	hashBytes := hash.Bytes()
	batch := store.db.NewBatch()

	// delete header, TD and receipts if any.
	headerKey := hashToHeaderKey(hashBytes)
	tdKey := hashToTDKey(hashBytes)
	receiptsKey := hashToReceiptsKey(hashBytes)
	if err := store.delete(batch, headerKey, tdKey, receiptsKey); err != nil {
		return err
	}

	// get body for more deletion
	bodyKey := hashToBodyKey(hashBytes)
	found, err := store.db.Has(bodyKey)
	if err != nil {
		return err
	}

	if !found {
		return batch.Commit()
	}

	encodedBody, err := store.db.Get(bodyKey)
	if err != nil {
		return err
	}

	var body blockBody
	if err = common.Deserialize(encodedBody, &body); err != nil {
		return err
	}

	// delete the tx/debt indices of the block.
	if err = store.batchDeleteIndices(batch, hash, body.Txs, body.Debts); err != nil {
		return err
	}

	// delete body
	batch.Delete(bodyKey)

	return batch.Commit()
}

// delete deletes data from the database given keys
func (store *blockchainDatabase) delete(batch database.Batch, keys ...[]byte) error {
	for _, k := range keys {
		found, err := store.db.Has(k)
		if err != nil {
			return err
		}

		if found {
			batch.Delete(k)
		}
	}

	return nil
}

// GetBlockByHeight gets the block with the specified height in the blockchain database
func (store *blockchainDatabase) GetBlockByHeight(height uint64) (*types.Block, error) {
	hash, err := store.GetBlockHash(height)
	if err != nil {
		return nil, err
	}
	block, err := store.GetBlock(hash)
	if err != nil {
		return nil, err
	}
	return block, nil
}

// PutReceipts serializes given receipts for the specified block hash.
func (store *blockchainDatabase) PutReceipts(hash common.Hash, receipts []*types.Receipt) error {
	encodedBytes, err := common.Serialize(receipts)
	if err != nil {
		return err
	}

	key := hashToReceiptsKey(hash.Bytes())

	return store.db.Put(key, encodedBytes)
}

// GetReceiptsByBlockHash retrieves the receipts for the specified block hash.
func (store *blockchainDatabase) GetReceiptsByBlockHash(hash common.Hash) ([]*types.Receipt, error) {
	key := hashToReceiptsKey(hash.Bytes())
	encodedBytes, err := store.db.Get(key)
	if err != nil {
		return nil, err
	}

	receipts := make([]*types.Receipt, 0)
	if err := common.Deserialize(encodedBytes, &receipts); err != nil {
		return nil, err
	}

	return receipts, nil
}

// GetReceiptByTxHash retrieves the receipt for the specified tx hash.
func (store *blockchainDatabase) GetReceiptByTxHash(txHash common.Hash) (*types.Receipt, error) {
	txIndex, err := store.GetTxIndex(txHash)
	if err != nil {
		return nil, err
	}

	receipts, err := store.GetReceiptsByBlockHash(txIndex.BlockHash)
	if err != nil {
		return nil, err
	}

	if uint(len(receipts)) <= txIndex.Index {
		return nil, fmt.Errorf("invalid tx index, txIndex = %v, receiptsLen = %v", *txIndex, len(receipts))
	}

	return receipts[txIndex.Index], nil
}

// PutDirtyAccounts serializes given dirty accounts for the specified block hash.
func (store *blockchainDatabase) PutDirtyAccounts(hash common.Hash, accounts []common.Address) error {
	encodedBytes, err := common.Serialize(accounts)
	if err != nil {
		return err
	}

	key := hashToDirtyAccountsKey(hash.Bytes())

	return store.db.Put(key, encodedBytes)
}

// GetDirtyAccountsByBlockHash retrieves the dirty accounts for the specified block hash.
func (store *blockchainDatabase) GetDirtyAccountsByBlockHash(hash common.Hash) ([]common.Address, error) {
	key := hashToDirtyAccountsKey(hash.Bytes())
	encodedBytes, err := store.db.Get(key)
	if err != nil {
		return nil, err
	}

	accounts := make([]common.Address, 0)
	if err := common.Deserialize(encodedBytes, &accounts); err != nil {
		return nil, err
	}

	return accounts, nil
}

// AddIndices adds tx/debt indices for the specified block.
func (store *blockchainDatabase) AddIndices(block *types.Block) error {
	batch := store.db.NewBatch()
	store.batchAddIndices(batch, block.HeaderHash, block.Transactions, block.Debts)
	return batch.Commit()
}

// batchAddIndices adds tx/debt indices to the blockchain database
func (store *blockchainDatabase) batchAddIndices(batch database.Batch, blockHash common.Hash, txs []*types.Transaction, debts []*types.Debt) {
	for i, tx := range txs {
		idx := types.TxIndex{BlockHash: blockHash, Index: uint(i)}
		batch.Put(txHashToIndexKey(tx.Hash.Bytes()), common.SerializePanic(idx))
	}

	for i, debt := range debts {
		idx := types.DebtIndex{BlockHash: blockHash, Index: uint(i)}
		batch.Put(debtHashToIndexKey(debt.Hash.Bytes()), common.SerializePanic(idx))
	}
}

// GetTxIndex retrieves the tx index for the specified tx hash.
func (store *blockchainDatabase) GetTxIndex(txHash common.Hash) (*types.TxIndex, error) {
	data, err := store.db.Get(txHashToIndexKey(txHash.Bytes()))
	if err != nil {
		return nil, err
	}

	index := &types.TxIndex{}
	if err := common.Deserialize(data, index); err != nil {
		return nil, err
	}

	return index, nil
}

// GetTxIndex retrieves the tx index for the specified tx hash.
func (store *blockchainDatabase) GetDebtIndex(debtHash common.Hash) (*types.DebtIndex, error) {
	data, err := store.db.Get(debtHashToIndexKey(debtHash.Bytes()))
	if err != nil {
		return nil, err
	}

	index := &types.DebtIndex{}
	if err := common.Deserialize(data, index); err != nil {
		return nil, err
	}

	return index, nil
}

// DeleteIndices deletes tx/debt indices of the specified block.
func (store *blockchainDatabase) DeleteIndices(block *types.Block) error {
	batch := store.db.NewBatch()

	if err := store.batchDeleteIndices(batch, block.HeaderHash, block.Transactions, block.Debts); err != nil {
		return err
	}

	return batch.Commit()
}

// batchDeleteIndices deletes tx/debt indices from the blockchain database
func (store *blockchainDatabase) batchDeleteIndices(batch database.Batch, blockHash common.Hash, txs []*types.Transaction, debts []*types.Debt) error {
	for _, tx := range txs {
		idx, err := store.GetTxIndex(tx.Hash)
		if err != nil {
			continue
		}

		if idx.BlockHash.Equal(blockHash) {
			batch.Delete(txHashToIndexKey(tx.Hash.Bytes()))
		}
	}

	for _, debt := range debts {
		idx, err := store.GetDebtIndex(debt.Hash)
		if err != nil {
			return err
		}

		if idx.BlockHash.Equal(blockHash) {
			batch.Delete(debtHashToIndexKey(debt.Hash.Bytes()))
		}
	}

	return nil
}
