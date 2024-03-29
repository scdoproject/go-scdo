/**
* @file
* @copyright defined in scdo/LICENSE
 */

package types

import (
	"fmt"
	"math/big"

	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/common/errors"
	"github.com/scdoproject/go-scdo/crypto"
	"github.com/scdoproject/go-scdo/trie"
)

// DebtSize debt serialized size
const DebtSize = 118

var (
	errWrongShardNumber  = errors.New("wrong from shard number")
	errInvalidAccount    = errors.New("invalid account, unexpected shard number")
	errInvalidHash       = errors.New("debt hash is invalid")
	errInvalidFee        = errors.New("debt fee is invalid")
	ErrMsgVerifierFailed = "failed to validate debt via verifier"
)

// DebtData debt data
type DebtData struct {
	TxHash  common.Hash // the hash of the executed transaction
	From    common.Address
	Nonce   uint64
	Account common.Address // debt for account
	Amount  *big.Int       // debt amount
	Price   *big.Int       // debt price
	Code    common.Bytes   // debt contract code
}

// Debt debt class
type Debt struct {
	Hash common.Hash // Debt hash of DebtData
	Data DebtData
}

// DebtIndex debt index
type DebtIndex indexInBlock

// GetDebtTrie generates a debt trie for the specified debts.
func GetDebtTrie(debts []*Debt) *trie.Trie {
	debtTrie := trie.NewEmptyTrie(make([]byte, 0), nil)

	for _, debt := range debts {
		if debt != nil {
			debtTrie.Put(debt.Hash.Bytes(), common.SerializePanic(debt))
		}
	}

	return debtTrie
}

// DebtMerkleRootHash calculates and returns the merkle root hash of the specified debts.
// If the given receipts are empty, return empty hash.
func DebtMerkleRootHash(debts []*Debt) common.Hash {
	debtTrie := GetDebtTrie(debts)
	return debtTrie.Hash()
}

// Validate validate debt with verifier
// If verifier is nil, will skip it.
// If isPool is true, we don't return error when the error is recoverable
func (d *Debt) Validate(verifier DebtVerifier, isPool bool, targetShard uint) (recoverable bool, retErr error) {
	if d.Data.From.Shard() == targetShard {
		retErr = errWrongShardNumber
		return
	}

	toShard := d.Data.Account.Shard()
	if targetShard != common.UndefinedShardNumber && toShard != targetShard {
		retErr = fmt.Errorf("invalid account, unexpected shard number, have %d, expected %d", toShard, targetShard)
		return
	}

	if d.Hash != d.Data.Hash() {
		retErr = errInvalidHash
		return
	}

	if d.Data.Price == nil || d.Data.Price.Sign() <= 0 {
		retErr = errInvalidFee
		return
	}

	// validate debt, skip validation when verifier is nil for test
	if verifier != nil {
		packed, confirmed, err := verifier.ValidateDebt(d)
		if packed {
			recoverable = true
		}

		if confirmed {
			return
		}

		if err != nil || !confirmed {
			if (isPool && !packed) || !isPool {
				retErr = errors.NewStackedError(err, ErrMsgVerifierFailed)
			}
		}
	}

	return
}

// Hash returns the hash of the debt data
func (data *DebtData) Hash() common.Hash {
	return crypto.MustHash(data)
}

// Size is the bytes of debt
func (d *Debt) Size() int {
	return DebtSize + len(d.Data.Code)
}

func (d *Debt) FromAccount() common.Address {
	return d.Data.From
}

func (d *Debt) ToAccount() common.Address {
	return d.Data.Account
}

func (d *Debt) Nonce() uint64 {
	return d.Data.Nonce
}

func (d *Debt) Price() *big.Int {
	return d.Data.Price
}

func (d *Debt) GetHash() common.Hash {
	return d.Hash
}

// GetDebtsSize is the bytes of debts
func GetDebtsSize(debts []*Debt) int {
	size := 0
	for _, d := range debts {
		size += d.Size()
	}

	return size
}

func (d *Debt) Fee() *big.Int {
	// @todo for contract case, should use the fee in tx receipt
	return new(big.Int).Mul(d.Data.Price, new(big.Int).SetUint64(DebtGas))
}

// NewDebtWithContext new a debt
func NewDebtWithContext(tx *Transaction) *Debt {
	return newDebt(tx, true)
}

// NewDebtWithoutContext new debt
func NewDebtWithoutContext(tx *Transaction) *Debt {
	return newDebt(tx, false)
}

// newDebt creates and returns a new debt from the given tx
func newDebt(tx *Transaction, withContext bool) *Debt {
	if tx == nil || tx.Data.To.IsEmpty() || tx.Data.To.IsReserved() {
		return nil
	}

	// reward transaction
	if tx.Data.From == common.EmptyAddress {
		return nil
	}

	toShard := tx.Data.To.Shard()
	if withContext && toShard == common.LocalShardNumber {
		return nil
	}

	fromShard := tx.Data.From.Shard()
	if !withContext && fromShard == toShard {
		return nil
	}

	data := DebtData{
		TxHash:  tx.Hash,
		From:    tx.Data.From,
		Nonce:   tx.Data.AccountNonce,
		Account: tx.Data.To,
		Amount:  big.NewInt(0).Set(tx.Data.Amount),
		Price:   tx.Data.GasPrice,
		Code:    make([]byte, 0), // @todo init when its a contract tx
	}

	if tx.Data.To.IsEVMContract() {
		data.Code = tx.Data.Payload
	}

	debt := &Debt{
		Data: data,
		Hash: data.Hash(),
	}

	return debt
}

// NewDebts new debts
func NewDebts(txs []*Transaction) []*Debt {
	debts := make([]*Debt, 0)

	for _, tx := range txs {
		d := NewDebtWithContext(tx)
		if d != nil {
			debts = append(debts, d)
		}
	}

	return debts
}

// NewDebtMap new debt map
func NewDebtMap(txs []*Transaction) [][]*Debt {
	debts := make([][]*Debt, common.ShardCount+1)

	for _, tx := range txs {
		d := NewDebtWithContext(tx)
		if d != nil {
			shard := d.Data.Account.Shard()
			debts[shard] = append(debts[shard], d)
		}
	}

	return debts
}

// DebtArrayToMap transfer debt array to debt map
func DebtArrayToMap(debts []*Debt) [][]*Debt {
	debtsMap := make([][]*Debt, common.ShardCount+1)

	for _, d := range debts {
		shard := d.Data.Account.Shard()
		debtsMap[shard] = append(debtsMap[shard], d)
	}

	return debtsMap
}

// BatchValidateDebt validates a batch of debts
func BatchValidateDebt(debts []*Debt, verifier DebtVerifier) error {
	return BatchValidate(func(index int) error {
		_, err := debts[index].Validate(verifier, false, common.LocalShardNumber)
		return err
	}, len(debts))
}
