/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package util

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/scdoproject/go-scdo/api"
	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/core/types"
	"github.com/scdoproject/go-scdo/rpc"
)

// GetAccountNonce get account nonce by account
func GetAccountNonce(client *rpc.Client, account common.Address, hexHash string, height int64) (uint64, error) {
	var nonce uint64
	// fmt.Println("PRINT GETACC:", account)
	err := client.Call(&nonce, "scdo_getAccountNonce", account, hexHash, height)
	return nonce, err
}

func GetInfo(client *rpc.Client) (api.GetMinerInfo, error) {
	var info api.GetMinerInfo
	err := client.Call(&info, "scdo_getInfo")

	return info, err
}

// GenerateTx generate a transaction based on the address type of to
func GenerateTx(from *ecdsa.PrivateKey, fromAddr *common.Address, to common.Address, amount *big.Int, price *big.Int, gasLimit uint64, nonce uint64, payload []byte) (*types.Transaction, error) {
	var err error

	var tx *types.Transaction
	if to.IsEmpty() {
		tx, err = types.NewContractTransaction(*fromAddr, amount, price, gasLimit, nonce, payload)
	} else {
		switch to.Type() {
		case common.AddressTypeExternal:
			// always ignore the user input gas limit for transfer amount tx.
			tx, err = types.NewTransaction(*fromAddr, to, amount, price, nonce)
		case common.AddressTypeContract, common.AddressTypeReserved:
			tx, err = types.NewMessageTransaction(*fromAddr, to, amount, price, gasLimit, nonce, payload)
		default:
			return nil, fmt.Errorf("unsupported address type: %d", to.Type())
		}
	}

	if err != nil {
		return nil, fmt.Errorf("create transaction err %s", err)
	}
	tx.Sign(from)

	return tx, nil
}

func GetTransactionByHash(client *rpc.Client, hash string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := client.Call(&result, "txpool_getTransactionByHash", hash)
	return result, err
}

func SendTx(client *rpc.Client, tx *types.Transaction) (bool, error) {
	var result bool
	err := client.Call(&result, "scdo_addTx", *tx)

	return result, err
}

// CallContract call contract
func CallContract(client *rpc.Client, contractID, payLoad string, height int64) (map[string]interface{}, error) {
	var info map[string]interface{}
	err := client.Call(&info, "scdo_call", contractID, payLoad, height)

	return info, err
}

// GetNetworkID get network ID
func GetNetworkID(client *rpc.Client) (string, error) {
	var networkID string
	err := client.Call(&networkID, "network_getNetworkID")

	return networkID, err
}
