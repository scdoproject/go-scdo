/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package p2p

import (
	"testing"

	"github.com/seelecredoteam/go-seelecredo/crypto"
	"github.com/seelecredoteam/go-seelecredo/p2p/discovery"
)

func getNode() *discovery.Node {
	return discovery.NewNode(*crypto.MustGenerateRandomAddress(), nil, 0, 1)
}

func Test_NodeSet(t *testing.T) {
	set := NewNodeSet()

	p1 := getNode()
	set.tryAdd(p1)

	p2 := set.randSelect()
	if p2 == nil {
		t.Fatalf("should select one node.")
	}

	set.delete(p2)
	if set.randSelect() != nil {
		t.Fatalf("should select no node.")
	}
}
