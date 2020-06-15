/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package trie

import (
	"testing"

	"github.com/scdoproject/go-scdo/crypto"
	"github.com/stretchr/testify/assert"
)

func Test_Node_DefaultValue(t *testing.T) {
	var n Node

	// default node is dirty with empty hash
	assert.Equal(t, nodeStatusDirty, n.Status())
	assert.Equal(t, []byte(nil), n.Hash())
}

func Test_Node_Update(t *testing.T) {
	var n Node

	// update status
	n.SetStatus(nodeStatusPersisted)
	assert.Equal(t, nodeStatusPersisted, n.Status())

	// update hash (node hash is nil)
	hash := crypto.MustHash("hello").Bytes()
	n.SetHash(hash)
	assert.Equal(t, hash, n.Hash())

	// update hash (node hash is not nil)
	hash = crypto.MustHash("world").Bytes()
	n.SetHash(hash)
	assert.Equal(t, hash, n.Hash())
}
