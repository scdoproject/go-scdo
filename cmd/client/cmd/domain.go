/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package cmd

import (
	"github.com/seeledevteam/slc/contract/system"
	"github.com/seeledevteam/slc/rpc"
)

// createDomainName create a domain name
func createDomainName(client *rpc.Client) (interface{}, interface{}, error) {
	amountValue = "0"
	if err := system.ValidateDomainName([]byte(nameValue)); err != nil {
		return nil, nil, err
	}

	tx, err := sendSystemContractTx(client, system.DomainNameContractAddress, system.CmdCreateDomainName, []byte(nameValue))
	if err != nil {
		return nil, nil, err
	}

	return tx, tx, err
}

// getDomainNameOwner get domain name owner
func getDomainNameOwner(client *rpc.Client) (interface{}, interface{}, error) {
	amountValue = "0"

	if err := system.ValidateDomainName([]byte(nameValue)); err != nil {
		return nil, nil, err
	}

	tx, err := sendSystemContractTx(client, system.DomainNameContractAddress, system.CmdGetDomainNameOwner, []byte(nameValue))
	if err != nil {
		return nil, nil, err
	}

	return tx, tx, err
}
