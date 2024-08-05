/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package api

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/common/errors"
	"github.com/scdoproject/go-scdo/common/hexutil"
	"github.com/scdoproject/go-scdo/core/state"
	"github.com/scdoproject/go-scdo/core/types"
)

// ErrInvalidAccount the account is invalid
var ErrInvalidAccount = errors.New("invalid account")

// maximum number of blocks to return in function GetBlocks
const maxSizeLimit = 64

// PublicScdoAPI provides an API to access full node-related information.
type PublicScdoAPI struct {
	s Backend
}

// NewPublicScdoAPI creates a new PublicScdoAPI object for rpc service.
func NewPublicScdoAPI(s Backend) *PublicScdoAPI {
	return &PublicScdoAPI{s}
}

// GetBalance get balance of the account.
func (api *PublicScdoAPI) GetBalance(account common.Address, hexHash string, height int64) (map[string]interface{}, error) {
	if account.IsEmpty() {
		return nil, ErrInvalidAccount
	}

	state, err := api.getStatedb(hexHash, height)
	if err != nil {
		return nil, errors.NewStackedError(err, "failed to get statedb")
	}

	var info GetBalanceResponse
	// is local shard?
	if common.LocalShardNumber != account.Shard() {
		return nil, fmt.Errorf("local shard is: %d, your shard is: %d, you need to change to shard %d to get your balance", common.LocalShardNumber, account.Shard(), account.Shard())
	}

	balance := state.GetBalance(account)
	if err = state.GetDbErr(); err != nil {
		return nil, errors.NewStackedError(err, "failed to get balance, db error occurred")
	}

	info.Balance = balance
	info.Account = account

	output := map[string]interface{}{
		"Balance": info.Balance,
		"Account": info.Account.Hex(),
	}
	return output, nil
}

// getStatedb gets the statedb of a block given the block hash or block height
func (api *PublicScdoAPI) getStatedb(hexHash string, height int64) (*state.Statedb, error) {
	var blockHash common.Hash
	var err error

	if len(hexHash) > 0 {
		if blockHash, err = common.HexToHash(hexHash); err != nil {
			return nil, errors.NewStackedError(err, "failed to convert HEX to hash")
		}
	} else if height < 0 {
		return api.s.ChainBackend().GetCurrentState()
	} else if blockHash, err = api.s.ChainBackend().GetStore().GetBlockHash(uint64(height)); err != nil {
		return nil, errors.NewStackedErrorf(err, "failed to get block hash by height %v", height)
	}

	header, err := api.s.ChainBackend().GetStore().GetBlockHeader(blockHash)
	if err != nil {
		return nil, errors.NewStackedErrorf(err, "failed to get block header by hash %v", blockHash)
	}

	return api.s.ChainBackend().GetState(header.StateHash)
}

// GetChangedAccounts gets the updated accounts of a certain block given the block hash or block height
func (api *PublicScdoAPI) GetChangedAccounts(hexHash string, height int64) (map[string]interface{}, error) {

	var blockHash common.Hash
	var err error

	if len(hexHash) > 0 {
		if blockHash, err = common.HexToHash(hexHash); err != nil {
			return nil, errors.NewStackedError(err, "failed to convert HEX to hash")
		}
	} else if height < 0 {
		return nil, errors.New("negative height")
	} else if blockHash, err = api.s.ChainBackend().GetStore().GetBlockHash(uint64(height)); err != nil {
		return nil, errors.NewStackedErrorf(err, "failed to get block hash by height %v", height)
	}

	accounts, err := api.s.ChainBackend().GetStore().GetDirtyAccountsByBlockHash(blockHash)
	if err != nil {
		return nil, err
	}

	var accountStrs []string
	for _, acc := range accounts {
		accountStrs = append(accountStrs, acc.Hex())
	}

	return map[string]interface{}{
		"blockHash":     blockHash,
		"account count": len(accounts),
		"accounts":      accountStrs,
	}, nil
}

// GetAccountNonce get account next used nonce
func (api *PublicScdoAPI) GetAccountNonce(account common.Address, hexHash string, height int64) (uint64, error) {
	if account.Equal(common.EmptyAddress) {
		return 0, ErrInvalidAccount
	}

	if common.LocalShardNumber != account.Shard() {
		return 0, fmt.Errorf("local shard is: %d, your shard is: %d, you need to change to shard %d to get your balance", common.LocalShardNumber, account.Shard(), account.Shard())
	}

	state, err := api.getStatedb(hexHash, height)
	if err != nil {
		return 0, err
	}
	nonce := state.GetNonce(account)
	if err = state.GetDbErr(); err != nil {
		return 0, err
	}
	var sourceNonce = nonce
	var pendingTxCount uint64
	// api.s.Log().Debug("pendingTx for account, %v, this nonce: %d", account, nonce)
	// get transactions from pending transactions, and plus nonce if its From address is current account
	pendingTxs := api.s.TxPoolBackend().GetTransactions(true, true)
	for _, tx := range pendingTxs {
		if tx.Data.From == account {
			// api.s.Log().Debug("pendingTx for account, %v", tx.Data.From)
			nonce++
			pendingTxCount++
		}
	}
	api.s.Log().Debug("pendingTx for account, %v, this sourceNonce: %d, this nonce: %d, pendingTxCount: %d", account, sourceNonce, nonce, pendingTxCount)
	return nonce, nil
}

// GetBlockHeight get the block height of the chain head
func (api *PublicScdoAPI) GetBlockHeight() (uint64, error) {
	header := api.s.ChainBackend().CurrentHeader()
	return header.Height, nil
}

// GetScdoForkHeight get the Scdo fork height
func (api *PublicScdoAPI) GetScdoForkHeight() (uint64, error) {
	return uint64(common.ScdoForkHeight), nil
}

// GetBlock returns the requested block.
func (api *PublicScdoAPI) GetBlock(hashHex string, height int64, fulltx bool) (map[string]interface{}, error) {
	if len(hashHex) > 0 {
		return api.GetBlockByHash(hashHex, fulltx)
	}

	return api.GetBlockByHeight(height, fulltx)
}

// GetBlockByHeight returns the requested block. When blockNr is less than 0 the chain head is returned. When fullTx is true all
// transactions in the block are returned in full detail, otherwise only the transaction hash is returned
func (api *PublicScdoAPI) GetBlockByHeight(height int64, fulltx bool) (map[string]interface{}, error) {
	block, err := api.s.GetBlock(common.EmptyHash, height)
	if err != nil {
		return nil, err
	}
	totalDifficulty, err := api.s.GetBlockTotalDifficulty(block.HeaderHash)
	if err != nil {
		return nil, err
	}
	return rpcOutputBlock(block, fulltx, totalDifficulty)
}

// GetBlocks returns requested blocks. When the blockNr is -1 the chain head is returned.
// When the size is greater than 64, the size will be set to 64.When it's -1 that the blockNr minus size, the blocks in 64 is returned.
// When fullTx is true all transactions in the block are returned in full detail, otherwise only the transaction hash is returned
func (api *PublicScdoAPI) GetBlocks(height int64, fulltx bool, size uint) ([]map[string]interface{}, error) {
	blocks := make([]*types.Block, 0)
	totalDifficultys := make([]*big.Int, 0)
	if height < 0 {
		header := api.s.ChainBackend().CurrentHeader()
		block, err := api.s.GetBlock(common.EmptyHash, int64(header.Height))
		if err != nil {
			return nil, err
		}
		totalDifficulty, err := api.s.GetBlockTotalDifficulty(block.HeaderHash)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
		totalDifficultys = append(totalDifficultys, totalDifficulty)
	} else {
		if size > maxSizeLimit {
			size = maxSizeLimit
		}

		if height+1-int64(size) < 0 {
			size = uint(height + 1)
		}

		for i := uint(0); i < size; i++ {
			var block *types.Block
			block, err := api.s.GetBlock(common.EmptyHash, height-int64(i))
			if err != nil {
				return nil, err
			}
			totalDifficulty, err := api.s.GetBlockTotalDifficulty(block.HeaderHash)
			if err != nil {
				return nil, err
			}
			totalDifficultys = append(totalDifficultys, totalDifficulty)
			blocks = append(blocks, block)
		}
	}

	return rpcOutputBlocks(blocks, fulltx, totalDifficultys)
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned
func (api *PublicScdoAPI) GetBlockByHash(hashHex string, fulltx bool) (map[string]interface{}, error) {
	hash, err := common.HexToHash(hashHex)
	if err != nil {
		return nil, err
	}

	block, err := api.s.GetBlock(hash, 0)
	if err != nil {
		return nil, err
	}

	totalDifficulty, err := api.s.GetBlockTotalDifficulty(block.HeaderHash)
	if err != nil {
		return nil, err
	}
	return rpcOutputBlock(block, fulltx, totalDifficulty)
}

// rpcOutputBlock converts the given block to the RPC output which depends on fullTx
func rpcOutputBlock(b *types.Block, fullTx bool, totalDifficulty *big.Int) (map[string]interface{}, error) {
	head := b.Header
	headmap := map[string]interface{}{
		"Consensus":         head.Consensus,
		"CreateTimestamp":   head.CreateTimestamp,
		"Creator":           head.Creator.Hex(),
		"DebtHash":          head.DebtHash,
		"Difficulty":        head.Difficulty,
		"ExtraData":         head.ExtraData,
		"Height":            head.Height,
		"PreviousBlockHash": head.PreviousBlockHash,
		"ReceiptHash":       head.ReceiptHash,
		"SecondWitness":     head.SecondWitness,
		"StateHash":         head.StateHash,
		"TxDebtHash":        head.TxDebtHash,
		"TxHash":            head.TxHash,
		"Witness":           head.Witness,
	}

	fields := map[string]interface{}{
		"header": headmap,
		"hash":   b.HeaderHash.Hex(),
	}

	txs := b.Transactions
	transactions := make([]interface{}, len(txs))
	for i, tx := range txs {
		if fullTx {
			transactions[i] = PrintableOutputTx(tx)
		} else {
			transactions[i] = tx.Hash.Hex()
		}
	}
	fields["transactions"] = transactions
	fields["totalDifficulty"] = totalDifficulty

	debts := types.NewDebts(txs)
	fields["txDebts"] = getOutputDebts(debts, fullTx)
	fields["debts"] = getOutputDebts(b.Debts, fullTx)

	return fields, nil
}

// getOutputDebts return the full details of the input debts if fullTx is true,
// otherwise only the hashes of the debts are returned
func getOutputDebts(debts []*types.Debt, fullTx bool) []interface{} {
	outputDebts := make([]interface{}, len(debts))
	for i, d := range debts {
		if fullTx {
			outputDebts[i] = d
		} else {
			outputDebts[i] = d.Hash
		}
	}

	return outputDebts
}

// rpcOutputBlocks converts the given blocks to the RPC output
func rpcOutputBlocks(b []*types.Block, fullTx bool, d []*big.Int) ([]map[string]interface{}, error) {
	fields := make([]map[string]interface{}, 0)

	for i := range b {
		if field, err := rpcOutputBlock(b[i], fullTx, d[i]); err == nil {
			fields = append(fields, field)
		}
	}
	return fields, nil
}

// PrintableOutputTx converts the given tx to the RPC output
func PrintableOutputTx(tx *types.Transaction) map[string]interface{} {
	toAddr := ""
	if !tx.Data.To.IsEmpty() {
		toAddr = tx.Data.To.Hex()
	}

	transaction := map[string]interface{}{
		"hash":         tx.Hash.Hex(),
		"from":         tx.Data.From.Hex(),
		"to":           toAddr,
		"amount":       tx.Data.Amount,
		"accountNonce": tx.Data.AccountNonce,
		"payload":      tx.Data.Payload,
		"gasPrice":     tx.Data.GasPrice,
		"gasLimit":     tx.Data.GasLimit,
		"signature":    tx.Signature,
	}
	return transaction
}

// AddTx add a tx to miner
func (api *PublicScdoAPI) AddTx(tx types.Transaction) (bool, error) {
	shard := tx.Data.From.Shard()
	var err error
	if shard != common.LocalShardNumber {
		if err = tx.ValidateWithoutState(true, false); err == nil {
			api.s.ProtocolBackend().SendDifferentShardTx(&tx, shard)
		}
	} else {
		err = api.s.TxPoolBackend().AddTransaction(&tx)
	}

	if err != nil {
		return false, err
	}
	api.s.Log().Debug("create transaction and add it. transaction hash: %v, time: %d", tx.Hash, time.Now().UnixNano())
	return true, nil
}

// GetCode gets the code of a contract address
func (api *PublicScdoAPI) GetCode(contractAdd common.Address, height int64) (interface{}, error) {
	state, err := api.getStatedb("", height)
	if err != nil {
		return nil, errors.NewStackedError(err, "failed to get statedb")
	}

	code := state.GetCode(contractAdd)
	return hexutil.BytesToHex(code), nil
}

// GetReceiptByTxHash get receipt by transaction hash
func (api *PublicScdoAPI) GetReceiptByTxHash(txHash, abiJSON string) (map[string]interface{}, error) {
	hash, err := common.HexToHash(txHash)
	if err != nil {
		return nil, err
	}

	receipt, err := api.s.GetReceiptByTxHash(hash)
	if err != nil {
		return nil, err
	}

	return printReceiptByABI(api, receipt, abiJSON)
}

// GetTransactionByBlockIndex returns the transaction in the block with the given block hash/height and index.
func (api *PublicScdoAPI) GetTransactionByBlockIndex(hashHex string, height int64, index uint) (map[string]interface{}, error) {
	if len(hashHex) > 0 {
		return api.GetTransactionByBlockHashAndIndex(hashHex, index)
	}

	return api.GetTransactionByBlockHeightAndIndex(height, index)
}

// GetTransactionByBlockHeightAndIndex returns the transaction in the block with the given block height and index.
func (api *PublicScdoAPI) GetTransactionByBlockHeightAndIndex(height int64, index uint) (map[string]interface{}, error) {
	block, err := api.s.GetBlock(common.EmptyHash, height)
	if err != nil {
		return nil, err
	}

	txs := block.Transactions
	if index >= uint(len(txs)) {
		return nil, errors.New("index out of block transaction list range, the max index is " + strconv.Itoa(len(txs)-1))
	}

	return PrintableOutputTx(txs[index]), nil
}

// GetTransactionByBlockHashAndIndex returns the transaction in the block with the given block hash and index.
func (api *PublicScdoAPI) GetTransactionByBlockHashAndIndex(hashHex string, index uint) (map[string]interface{}, error) {
	hash, err := common.HexToHash(hashHex)
	if err != nil {
		return nil, err
	}

	block, err := api.s.GetBlock(hash, 0)
	if err != nil {
		return nil, err
	}

	txs := block.Transactions
	if index >= uint(len(txs)) {
		return nil, errors.New("index out of block transaction list range, the max index is " + strconv.Itoa(len(txs)-1))
	}

	return PrintableOutputTx(txs[index]), nil
}

// GetBlockTransactionCount returns the count of transactions in the block with the given block hash or height.
func (api *PublicScdoAPI) GetBlockTransactionCount(blockHash string, height int64) (int, error) {
	if len(blockHash) > 0 {
		return api.GetBlockTransactionCountByHash(blockHash)
	}

	return api.GetBlockTransactionCountByHeight(height)
}

// GetBlockDebtCount returns the count of debts in the block with the given block hash or height.
func (api *PublicScdoAPI) GetBlockDebtCount(blockHash string, height int64) (int, error) {
	if len(blockHash) > 0 {
		return api.GetBlockDebtCountByHash(blockHash)
	}

	return api.GetBlockDebtCountByHeight(height)
}

// GetBlockTransactionCountByHeight returns the count of transactions in the block with the given height.
func (api *PublicScdoAPI) GetBlockTransactionCountByHeight(height int64) (int, error) {
	block, err := api.s.GetBlock(common.EmptyHash, height)
	if err != nil {
		return 0, err
	}

	return len(block.Transactions), nil
}

// GetBlockTransactionCountByHash returns the count of transactions in the block with the given hash.
func (api *PublicScdoAPI) GetBlockTransactionCountByHash(blockHash string) (int, error) {
	hash, err := common.HexToHash(blockHash)
	if err != nil {
		return 0, err
	}

	block, err := api.s.GetBlock(hash, 0)
	if err != nil {
		return 0, err
	}

	return len(block.Transactions), nil
}

// GetBlockDebtCountByHeight returns the count of debts in the block with the given height.
func (api *PublicScdoAPI) GetBlockDebtCountByHeight(height int64) (int, error) {
	block, err := api.s.GetBlock(common.EmptyHash, height)
	if err != nil {
		return 0, err
	}

	return len(block.Debts), nil
}

// GetBlockDebtCountByHash returns the count of debts in the block with the given hash.
func (api *PublicScdoAPI) GetBlockDebtCountByHash(blockHash string) (int, error) {
	hash, err := common.HexToHash(blockHash)
	if err != nil {
		return 0, err
	}

	block, err := api.s.GetBlock(hash, 0)
	if err != nil {
		return 0, err
	}

	return len(block.Debts), nil
}

// GetReceiptsByBlockHash get receipts by block hash
func (api *PublicScdoAPI) GetReceiptsByBlockHash(blockHash string) (map[string]interface{}, error) {
	hash, err := common.HexToHash(blockHash)
	if err != nil {
		return nil, err
	}

	receipts, err := api.s.ChainBackend().GetStore().GetReceiptsByBlockHash(hash)
	if err != nil {
		return nil, err
	}

	outMaps := make([]map[string]interface{}, 0, len(receipts))
	for _, re := range receipts {
		outMap, err := PrintableReceipt(re)
		if err != nil {
			return nil, err
		}
		outMaps = append(outMaps, outMap)
	}

	return map[string]interface{}{
		"blockHash": blockHash,
		"receipts":  outMaps,
	}, nil
}

// IsSyncing returns the sync status of the node
func (api *PublicScdoAPI) IsSyncing() bool {
	return api.s.IsSyncing()
}

// Always listening
func (api *PublicScdoAPI) IsListening() bool { return true }

// GetTransactionsTo get transactions from one account at specific height or blockhash
func (api *PublicScdoAPI) GetTransactionsFrom(account common.Address, blockHash string, height int64) (result []map[string]interface{}, err error) {
	if len(blockHash) > 0 {
		return api.GetTransactionsFromByHash(account, blockHash)
	}
	return api.GetTransactionsFromByHeight(account, height)
}

// GetTransactionsTo get transactions to one account at specific height or blockhash
func (api *PublicScdoAPI) GetTransactionsTo(account common.Address, blockHash string, height int64) (result []map[string]interface{}, err error) {
	if len(blockHash) > 0 {
		return api.GetTransactionsToByHash(account, blockHash)
	}
	return api.GetTransactionsToByHeight(account, height)
}

// GetTransactionsFromByHash get transaction from one account at specific blockhash
func (api *PublicScdoAPI) GetTransactionsFromByHash(account common.Address, blockHash string) (result []map[string]interface{}, err error) {
	var txCount = 0
	hash, err := common.HexToHash(blockHash)
	if err != nil {
		return nil, err
	}
	block, err := api.s.GetBlock(hash, 0)
	if err != nil {
		return nil, err
	}
	txs := block.Transactions
	for _, tx := range txs {
		if tx.FromAccount() == account {
			txCount++
			output := map[string]interface{}{
				"transaction" + fmt.Sprintf(" %d", txCount): PrintableOutputTx(tx),
			}
			result = append(result, output)
		}
	}

	return result, nil
}

// GetTransactionsToByHash get transaction from one account at specific blockhash
func (api *PublicScdoAPI) GetTransactionsToByHash(account common.Address, blockHash string) (result []map[string]interface{}, err error) {
	var txCount = 0
	hash, err := common.HexToHash(blockHash)
	if err != nil {
		return nil, err
	}
	block, err := api.s.GetBlock(hash, 0)
	if err != nil {
		return nil, err
	}
	txs := block.Transactions
	for _, tx := range txs {
		if tx.ToAccount() == account {
			txCount++
			output := map[string]interface{}{
				"transaction" + fmt.Sprintf(" %d", txCount): PrintableOutputTx(tx),
			}
			result = append(result, output)
		}
	}

	return result, nil
}

// GetTransactionsFromByHeight get transaction from one account at specific height
func (api *PublicScdoAPI) GetTransactionsFromByHeight(account common.Address, height int64) (result []map[string]interface{}, err error) {
	var txCount = 0
	block, err := api.s.GetBlock(common.EmptyHash, height)
	if err != nil {
		return nil, err
	}
	txs := block.Transactions
	for _, tx := range txs {
		if tx.FromAccount() == account {
			txCount++
			output := map[string]interface{}{
				"transaction" + fmt.Sprintf(" %d", txCount): PrintableOutputTx(tx),
			}
			result = append(result, output)

		}
	}
	return result, nil
}

// GetTransactionsToByHeight get transaction from one account at specific height
func (api *PublicScdoAPI) GetTransactionsToByHeight(account common.Address, height int64) (result []map[string]interface{}, err error) {
	var txCount = 0
	block, err := api.s.GetBlock(common.EmptyHash, height)
	if err != nil {
		return nil, err
	}
	txs := block.Transactions
	for _, tx := range txs {
		if tx.ToAccount() == account {
			txCount++
			output := map[string]interface{}{
				"transaction" + fmt.Sprintf(" %d", txCount): PrintableOutputTx(tx),
			}
			result = append(result, output)
		}
	}

	return result, nil
}

// GetAccountTransactions get transaction of one account at specific height or blockhash
func (api *PublicScdoAPI) GetAccountTransactions(account common.Address, blockHash string, height int64) (result []map[string]interface{}, err error) {
	if len(blockHash) > 0 {
		return api.GetAccountTransactionsByHash(account, blockHash)
	}
	return api.GetAccountTransactionsByHeight(account, height)
}

// GetAccountTransactionsByHash get transaction of one account at specific height
func (api *PublicScdoAPI) GetAccountTransactionsByHash(account common.Address, blockHash string) (result []map[string]interface{}, err error) {
	hash, err := common.HexToHash(blockHash)
	if err != nil {
		return nil, err
	}
	block, err := api.s.GetBlock(hash, 0)
	if err != nil {
		return nil, err
	}
	txs := block.Transactions
	for i, tx := range txs {
		if tx.FromAccount() == account || tx.ToAccount() == account {
			output := map[string]interface{}{
				"transaction" + fmt.Sprintf(" %d", i): PrintableOutputTx(tx),
			}
			result = append(result, output)
		}
	}
	return result, nil
}

// GetAccountTransactionsByHeight get transaction of one account at specific blockhash
func (api *PublicScdoAPI) GetAccountTransactionsByHeight(account common.Address, height int64) (result []map[string]interface{}, err error) {
	block, err := api.s.GetBlock(common.EmptyHash, height)
	if err != nil {
		return nil, err
	}
	txs := block.Transactions
	for i, tx := range txs {
		if tx.FromAccount() == account || tx.ToAccount() == account {
			output := map[string]interface{}{
				"transaction" + fmt.Sprintf(" %d", i): PrintableOutputTx(tx),
			}
			result = append(result, output)
		}
	}
	return result, nil
}

// GetBlockTransactions get all txs in the block with height or blockhash
func (api *PublicScdoAPI) GetBlockTransactions(blockHash string, height int64) (result []map[string]interface{}, err error) {
	if len(blockHash) > 0 {
		return api.GetBlockTransactionsByHash(blockHash)
	}

	return api.GetBlockTransactionsByHeight(height)
}

// GetBlockTransactionsByHeight returns the transactions in the block with the given height.
func (api *PublicScdoAPI) GetBlockTransactionsByHeight(height int64) (result []map[string]interface{}, err error) {
	block, err := api.s.GetBlock(common.EmptyHash, height)
	if err != nil {
		return nil, err
	}
	txs := block.Transactions
	for i, tx := range txs {
		output := map[string]interface{}{
			"transaction" + fmt.Sprintf(" %d", i+1): PrintableOutputTx(tx),
		}
		result = append(result, output)
	}
	return result, nil
}

// GetBlockTransactionsByHash returns the transactions in the block with the given height.
func (api *PublicScdoAPI) GetBlockTransactionsByHash(blockHash string) (result []map[string]interface{}, err error) {
	hash, err := common.HexToHash(blockHash)
	if err != nil {
		return nil, err
	}

	block, err := api.s.GetBlock(hash, 0)
	if err != nil {
		return nil, err
	}
	txs := block.Transactions
	for i, tx := range txs {
		output := map[string]interface{}{
			"transaction" + fmt.Sprintf(" %d", i+1): PrintableOutputTx(tx),
		}
		result = append(result, output)
	}
	return result, nil
}
