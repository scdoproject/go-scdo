package api

import (
	"encoding/json"
	"strings"

	"github.com/scdoproject/go-scdo/accounts/abi"
	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/core/types"
)

// KeyABIHash is the hash key to storing abi to statedb
var KeyABIHash = common.StringToHash("KeyABIHash")

type scdoLog struct {
	Topics []string
	Event  string
	Args   []interface{}
}

func printReceiptByABI(api *PublicScdoAPI, receipt *types.Receipt, abiJSON string) (map[string]interface{}, error) {
	result, err := PrintableReceipt(receipt)
	if err != nil {
		return nil, err
	}

	// unpack result - todo: Since the methodName cannot be found now, it will be parsed in the next release.

	// unpack log
	if len(receipt.Logs) > 0 {
		logOuts := make([]string, 0)

		for _, log := range receipt.Logs {
			parsed, err := abi.JSON(strings.NewReader(abiJSON))
			if err != nil {
				api.s.Log().Warn("invalid abiJSON '%s', err: %s", abiJSON, err)
				return result, nil
			}

			logOut, err := printLogByABI(log, parsed)
			if err != nil {
				api.s.Log().Warn("err: %s", err)
				return result, nil
			}

			logOuts = append(logOuts, logOut)
		}

		result["logs"] = logOuts
	}

	return result, nil
}

func printLogByABI(log *types.Log, parsed abi.ABI) (string, error) {
	scdolog := &scdoLog{}
	if len(log.Topics) < 1 {
		return "", nil
	}

	for _, topic := range log.Topics {
		scdolog.Topics = append(scdolog.Topics, topic.Hex())
	}

	for _, event := range parsed.Events {
		if event.Id().Hex() == scdolog.Topics[0] {
			scdolog.Event = event.Name
			break
		}
	}

	var err error
	scdolog.Args, err = parsed.Events[scdolog.Event].Inputs.UnpackValues(log.Data)
	if err != nil {
		return "", err
	}

	encoded, err := json.Marshal(scdolog)
	if err != nil {
		return "", err
	}

	return string(encoded), nil
}
