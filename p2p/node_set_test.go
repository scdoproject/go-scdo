/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package p2p

import (
	"testing"

	"github.com/scdoproject/go-scdo/crypto"
	"github.com/scdoproject/go-scdo/p2p/discovery"
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
