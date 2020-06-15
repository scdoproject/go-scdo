/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package light

import (
	"testing"

	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/database/leveldb"
	"github.com/scdoproject/go-scdo/trie"
	"github.com/stretchr/testify/assert"
)

type mockOdrRetriever struct {
	resp odrResponse
}

func (r *mockOdrRetriever) retrieveWithFilter(request odrRequest, filter peerFilter) (odrResponse, error) {
	return r.resp, nil
}

func Test_Trie_Get(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	// prepare trie on server side
	dbPrefix := []byte("test prefix")
	trie := trie.NewEmptyTrie(dbPrefix, db)
	trie.Put([]byte("hello"), []byte("HELLO"))
	trie.Put([]byte("scdo"), []byte("SCDO"))
	trie.Put([]byte("world"), []byte("WORLD"))

	// prepare mock odr retriever
	proof, err := trie.GetProof([]byte("scdo"))
	assert.Nil(t, err)
	retriever := &mockOdrRetriever{
		resp: &odrTriePoof{
			Proof: mapToArray(proof),
		},
	}

	// validate on light client
	lightTrie := newOdrTrie(retriever, trie.Hash(), dbPrefix, common.EmptyHash)

	// key exists
	v, ok, err := lightTrie.Get([]byte("scdo"))
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, []byte("SCDO"), v)

	// key not found
	v, ok, err = lightTrie.Get([]byte("scdo 2"))
	assert.Nil(t, err)
	assert.False(t, ok)
	assert.Nil(t, v)
}
