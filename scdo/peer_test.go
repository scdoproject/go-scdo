/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package scdo

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/crypto"
	log2 "github.com/scdoproject/go-scdo/log"
	"github.com/scdoproject/go-scdo/p2p"
	"github.com/scdoproject/go-scdo/p2p/discovery"
	"github.com/stretchr/testify/assert"
)

func Test_peer_Info(t *testing.T) {
	// prepare some variables
	myAddr := *crypto.MustGenerateShardAddress(1)
	node1 := discovery.NewNode(myAddr, nil, 0, 0)
	log := log2.GetLogger("test")
	p2pPeer := &p2p.Peer{
		Node: node1,
	}
	var myHash common.Hash
	copy(myHash[0:20], myAddr[:])
	bigInt := big.NewInt(100)
	okStr := fmt.Sprintf(`{"version":1,"difficulty":100,"head":"%v000000000000000000000000"}`, strings.TrimPrefix(myAddr.Hex(), "0x"))

	// Create peer for test
	peer := newPeer(common.ScdoVersion, p2pPeer, nil, log)
	peer.SetHead(myHash, bigInt)

	peerInfo := peer.Info()
	data, _ := json.Marshal(peerInfo)
	resultStr := string(data)
	if okStr != resultStr {
		t.Fail()
	}
}

func Test_verifyGenesis(t *testing.T) {
	networkID := "scdo"
	statusData := statusData{
		ProtocolVersion: uint32(0),
		NetworkID:       networkID,
		TD:              big.NewInt(0),
		CurrentBlock:    common.EmptyHash,
		GenesisBlock:    common.EmptyHash,
		Shard:           1,
		Difficult:       8000000,
	}
	err := verifyGenesisAndNetworkID(statusData, common.EmptyHash, networkID, 1, 8000000)
	assert.Equal(t, err, nil)

	err = verifyGenesisAndNetworkID(statusData, common.EmptyHash, networkID, 2, 8000000)
	assert.Equal(t, err, nil)

	err = verifyGenesisAndNetworkID(statusData, common.EmptyHash, networkID, 2, 9000000)
	assert.Equal(t, err == errGenesisDifficultNotMatch, true)

	errorHash := common.StringToHash("error hash")
	err = verifyGenesisAndNetworkID(statusData, errorHash, networkID, 1, 8000000)
	assert.Equal(t, err != nil, true)
}
