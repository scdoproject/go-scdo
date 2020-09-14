/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package cmd

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/scdoproject/go-scdo/cmd/util"
	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/common/hexutil"
	"github.com/scdoproject/go-scdo/core/types"
	"github.com/scdoproject/go-scdo/crypto"
	"github.com/scdoproject/go-scdo/rpc"
	"github.com/urfave/cli"
)

const (
	// DefaultNonce is the default value of nonce,when you are not set the nonce flag in client sendtx command by --nonce .
	DefaultNonce uint64 = 0
)

func checkParameter(publicKey *ecdsa.PublicKey, client *rpc.Client, keyaddress common.Address) (*types.TransactionData, error) {
	info := &types.TransactionData{}
	var err error
	if len(toValue) > 0 {
		toAddr, err := common.HexToAddress(toValue)
		if err != nil {
			return info, fmt.Errorf("invalid receiver address: %s", err)
		}
		info.To = toAddr
	}

	amount, ok := big.NewInt(0).SetString(amountValue, 10)
	if !ok {
		return info, fmt.Errorf("invalid amount value")
	}
	info.Amount = amount

	price, ok := big.NewInt(0).SetString(priceValue, 10)
	if !ok {
		return info, fmt.Errorf("invalid gas price value")
	}
	info.GasPrice = price

	info.GasLimit = gasLimitValue

	fromAddr, err := crypto.GetAddress(publicKey, shardValue)

	fmt.Println(keyaddress)
	if keyaddress.IsEmpty() {
		info.From = *fromAddr
	} else {
		info.From = keyaddress
		if keyaddress.Shard() != shardValue {
			modAddr, err := crypto.GetAddress(publicKey, shardValue)
			if err != nil {
				return info, fmt.Errorf("invalid shard num")
			}
			info.From = *modAddr
		}
	}

	if nonceValue == DefaultNonce && client != nil {
		// get current nonce
		nonce, err := util.GetAccountNonce(client, info.From, "", -1)
		if err != nil {
			return info, fmt.Errorf("failed to get the sender account's nonce: %s", err)
		}
		info.AccountNonce = nonce
		fmt.Printf("sendtx without setting nonce, GetAccountNonce %d\n", nonce)
	} else {
		// get current nonce
		dbnonce, nonceErr := util.GetAccountNonce(client, info.From, "", -1)
		if nonceErr != nil {
			return info, fmt.Errorf("failed to get the sender account nonce: %s", err)
		}
		if nonceValue < dbnonce {
			return info, fmt.Errorf("Failed to sendtx for setNonce %d is smaller than database nonce %d\n", nonceValue, dbnonce)
		}
		info.AccountNonce = nonceValue
	}
	fmt.Printf("account: %s, transaction nonce: %d\n", info.From.Hex(), info.AccountNonce)

	payload := []byte(nil)
	if len(payloadValue) > 0 {
		if payload, err = hexutil.HexToBytes(payloadValue); err != nil {
			return info, fmt.Errorf("invalid payload, %s", err)
		}
	}
	info.Payload = payload
	return info, nil
}

// NewApp generate default app
func NewApp(isFullNode bool) *cli.App {
	app := cli.NewApp()
	app.Usage = addUsage(isFullNode)
	app.HideVersion = true
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "scdoteamteam",
			Email: "dev@scdonet.com",
		},
	}

	AddCommands(app, isFullNode)
	return app
}

func addUsage(isFullNode bool) string {
	if isFullNode {
		return "interact with a full node process"
	}

	return "interact with a light node process"
}
