/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package scdo

import (
	"fmt"
	"strings"

	"github.com/scdoproject/go-scdo/accounts/abi"
	"github.com/scdoproject/go-scdo/accounts/abi/bind"
	"github.com/scdoproject/go-scdo/common/hexutil"
)

// GeneratePayload according to abi json string and methodName and args to generate payload hex string
func (api *PublicScdoAPI) GeneratePayload(abiJSON string, methodName string, args []string) (string, error) {
	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return "", fmt.Errorf("invalid abiJSON '%s', err: %s", abiJSON, err)
	}

	method, exist := parsed.Methods[methodName]
	if !exist {
		return "", fmt.Errorf("method '%s' not found", methodName)
	}

	seeleTypeArgs, err := bind.ParseArgs(method.Inputs, args)
	if err != nil {
		return "", err
	}

	bytes, err := parsed.Pack(methodName, seeleTypeArgs...)
	if err != nil {
		return "", err
	}

	return hexutil.BytesToHex(bytes), nil
}
