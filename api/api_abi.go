package api

import (
	"encoding/json"
	"strings"

	"github.com/seelecredo/go-seelecredo/accounts/abi"
	"github.com/seelecredo/go-seelecredo/common"
	"github.com/seelecredo/go-seelecredo/core/types"
)

// KeyABIHash is the hash key to storing abi to statedb
var KeyABIHash = common.StringToHash("KeyABIHash")

type seeleCredoLog struct {
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
	seeleCredolog := &seeleCredoLog{}
	if len(log.Topics) < 1 {
		return "", nil
	}

	for _, topic := range log.Topics {
		seeleCredolog.Topics = append(seeleCredolog.Topics, topic.Hex())
	}

	for _, event := range parsed.Events {
		if event.Id().Hex() == seeleCredolog.Topics[0] {
			seeleCredolog.Event = event.Name
			break
		}
	}

	var err error
	seeleCredolog.Args, err = parsed.Events[seeleCredolog.Event].Inputs.UnpackValues(log.Data)
	if err != nil {
		return "", err
	}

	encoded, err := json.Marshal(seeleCredolog)
	if err != nil {
		return "", err
	}

	return string(encoded), nil
}
