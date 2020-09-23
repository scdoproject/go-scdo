/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package scdo

import (
	api2 "github.com/scdoproject/go-scdo/api"
	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/common/hexutil"
	"github.com/scdoproject/go-scdo/core/types"
)

// TransactionPoolAPI provides an API to access transaction pool information.
type TransactionPoolAPI struct {
	s *ScdoService
}

// NewTransactionPoolAPI creates a new PrivateTransactionPoolAPI object for transaction pool rpc service.
func NewTransactionPoolAPI(s *ScdoService) *TransactionPoolAPI {
	return &TransactionPoolAPI{s}
}

// GetPendingDebts returns all pending debts
func (api *TransactionPoolAPI) GetPendingDebts() ([]*types.Debt, error) {
	return api.s.DebtPool().GetDebts(false, true), nil
}

// GetDebtByHash return the debt info by debt hash
func (api *TransactionPoolAPI) GetDebtByHash(debtHash string) (map[string]interface{}, error) {
	hashByte, err := hexutil.HexToBytes(debtHash)
	if err != nil {
		return nil, err
	}
	hash := common.BytesToHash(hashByte)

	debt, blockIdx, err := api2.GetDebt(api.s.DebtPool(), api.s.chain.GetStore(), hash)
	if err != nil {
		return nil, err
	}
	debtData := map[string]interface{}{
		"Account": debt.Data.Account.Hex(),
		"Amount":  debt.Data.Amount,
		"Code":    debt.Data.Code,
		"From":    debt.Data.From.Hex(),
		"Nonce":   debt.Data.Nonce,
		"Price":   debt.Data.Price,
		"TxHash":  debt.Data.TxHash,
	}
	debtOutput := map[string]interface{}{
		"Hash": debt.Hash,
		"Data": debtData,
	}

	output := map[string]interface{}{
		"debt": debtOutput,
	}

	if blockIdx == nil {
		output["status"] = "pool"
	} else {
		output["status"] = "block"
		output["blockHash"] = blockIdx.BlockHash.Hex()
		output["blockHeight"] = blockIdx.BlockHeight
		output["debtIndex"] = blockIdx.Index
	}

	return output, nil
}
