/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package util

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/common/hexutil"
	"github.com/scdoproject/go-scdo/crypto"
	"github.com/spf13/cobra"
)

// GetGenerateKeyPairCmd represents the generateKeyPair command
func GetGenerateKeyPairCmd(name string) (cmds *cobra.Command) {
	var shard *uint

	var generateKeyPairCmd = &cobra.Command{
		Use:   "key",
		Short: "generate a key pair with specified shard number",
		Long:  "generate a key pair and print them with hex values\n For example:\n" + name + " key --shard 1",
		Run: func(cmd *cobra.Command, args []string) {
			publicKey, privateKey, err := GenerateKey(*shard)
			if err != nil {
				fmt.Println(err)
				return
			}

			fmt.Printf("Account:  %s\n", publicKey.Hex())
			fmt.Printf("private key: %s\n", hexutil.BytesToHex(crypto.FromECDSA(privateKey)))
		},
	}

	shard = generateKeyPairCmd.Flags().UintP("shard", "", 0, "shard number")

	return generateKeyPairCmd
}

// GenerateKey generate key by shard
func GenerateKey(shard uint) (*common.Address, *ecdsa.PrivateKey, error) {
	var publicKey *common.Address
	var privateKey *ecdsa.PrivateKey
	var err error
	if shard > common.ShardCount {
		return nil, nil, fmt.Errorf("not supported shard number, shard number should be [0, %d]", common.ShardCount)
	} else if shard == 0 {
		shard := crypto.RandomShard()
		publicKey, privateKey, err = crypto.GenerateKeyPair(shard)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate the key pair: %s", err)
		}
	} else {
		publicKey, privateKey = crypto.MustGenerateShardKeyPair(shard)
	}

	return publicKey, privateKey, nil
}
