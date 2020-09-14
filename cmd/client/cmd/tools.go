/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/scdoproject/go-scdo/accounts/abi"
	"github.com/scdoproject/go-scdo/accounts/abi/bind"
	"github.com/scdoproject/go-scdo/api"
	"github.com/scdoproject/go-scdo/cmd/util"
	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/common/hexutil"
	"github.com/scdoproject/go-scdo/common/keystore"
	"github.com/scdoproject/go-scdo/core/types"
	"github.com/scdoproject/go-scdo/crypto"
	"github.com/scdoproject/go-scdo/rpc"
	"github.com/urfave/cli"
)

// GetAccountShardNumAction is a action to get the shard number of account
func GetAccountShardNumAction(c *cli.Context) error {
	var accountAddress common.Address
	fmt.Println("s.GetAccountShardNumAction:", accountValue)
	address, err := common.HexToAddress(accountValue)
	if err != nil {
		return fmt.Errorf("the account is invalid for: %v", err)
	}

	accountAddress = address

	shard := accountAddress.Shard()
	fmt.Printf("shard number: %d\n", shard)
	return nil
}

// SaveKeyAction is a action to save the private key to the file
func SaveKeyAction(c *cli.Context) error {
	privateKey, err := crypto.LoadECDSAFromString(privateKeyValue)
	if err != nil {
		return fmt.Errorf("invalid key: %s", err)
	}

	if fileNameValue == "" {
		return fmt.Errorf("please specify the key file path")
	}

	if !common.ValidShard(shardValue) {
		return fmt.Errorf("Invalid shard num %v", shardValue)
	}

	pass, err := common.SetPassword()
	if err != nil {
		return fmt.Errorf("get password err %s", err)
	}
	addr, err := crypto.GetAddress(&privateKey.PublicKey, shardValue)
	key := keystore.Key{
		Address:    *addr,
		PrivateKey: privateKey,
	}

	err = keystore.StoreKey(fileNameValue, pass, &key)
	if err != nil {
		return fmt.Errorf("failed to store the key file %s, %s", fileNameValue, err.Error())
	}

	fmt.Printf("store key successfully, the key file path is %s\n", fileNameValue)
	return nil
}

// SignTxAction is a action that signs a transaction
func SignTxAction(c *cli.Context) error {
	var client *rpc.Client
	if addressValue != "" {
		c, err := rpc.DialTCP(context.Background(), addressValue)
		if err != nil {
			return err
		}

		client = c
	}

	key, err := crypto.LoadECDSAFromString(privateKeyValue)
	if err != nil {
		return fmt.Errorf("failed to load key %s", err)
	}

	txd, err := checkParameter(&key.PublicKey, client, common.EmptyAddress)
	if err != nil {
		return err
	}

	var tx = types.Transaction{}
	tx.Data = *txd
	tx.Sign(key)

	output := map[string]interface{}{
		"Transaction": api.PrintableOutputTx(&tx),
	}
	result, err := json.MarshalIndent(output, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(result))

	return nil
}

// GenerateKeyAction generate key by client command
func GenerateKeyAction(c *cli.Context) error {
	publicKey, privateKey, err := util.GenerateKey(shardValue)
	if err != nil {
		return err
	}

	fmt.Printf("Account:  %s\n", publicKey.Hex())
	fmt.Printf("Private key: %s\n", hexutil.BytesToHex(crypto.FromECDSA(privateKey)))
	return nil
}

// DecryptKeyFileAction decrypt key file
func DecryptKeyFileAction(c *cli.Context) error {
	if fileNameValue == "" {
		return fmt.Errorf("Filename empty")
	}

	pass, err := common.GetPassword()
	if err != nil {
		return fmt.Errorf("failed to get password %s", err)
	}

	key, err := keystore.GetKey(fileNameValue, pass)
	if err != nil {
		return fmt.Errorf("invalid key file: %s", err)
	}

	fmt.Printf("Account:  %s\n", key.Address.Hex())
	fmt.Printf("Private key: %s\n", hexutil.BytesToHex(crypto.FromECDSA(key.PrivateKey)))
	return nil
}

// GeneratePayloadAction is a action to generate the payload according to the abi string and method name and args
func GeneratePayloadAction(c *cli.Context) error {
	if abiFile == "" || methodName == "" {
		return fmt.Errorf("required flag(s) \"abi, method\" not set")
	}

	abiJSON, err := readABIFile(abiFile)
	if err != nil {
		return err
	}

	payload, err := generatePayload(abiJSON, methodName, c.StringSlice("args"))
	if err != nil {
		return fmt.Errorf("failed to parse the abi, err:%s", err)
	}

	fmt.Printf("payload: %s\n", hexutil.BytesToHex(payload))
	return nil
}

func generatePayload(abiStr, methodName string, args []string) ([]byte, error) {
	parsed, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		return nil, fmt.Errorf("failed to parse the abi, err:%s", err)
	}

	method, exist := parsed.Methods[methodName]
	if !exist {
		return nil, fmt.Errorf("method '%s' not found", methodName)
	}

	ss, err := bind.ParseArgs(method.Inputs, args)
	if err != nil {
		return nil, err
	}

	return parsed.Pack(methodName, ss...)
}

func readABIFile(abiFile string) (string, error) {
	if !common.FileOrFolderExists(abiFile) {
		return "", fmt.Errorf("The specified abi file[%s] does not exist", abiFile)
	}

	bytes, err := ioutil.ReadFile(abiFile)
	if err != nil {
		return "", fmt.Errorf("failed to read abi file, err: %s", err)
	}

	return string(bytes), nil
}
