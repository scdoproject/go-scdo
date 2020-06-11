/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package seeleCredo

import (
	//"context"
	"context"
	"path/filepath"
	"testing"

	"github.com/seelecredo/go-seelecredo/common"
	"github.com/seelecredo/go-seelecredo/consensus/factory"
	"github.com/seelecredo/go-seelecredo/core"
	"github.com/seelecredo/go-seelecredo/crypto"
	"github.com/seelecredo/go-seelecredo/log"
	"github.com/seelecredo/go-seelecredo/node"
	"github.com/stretchr/testify/assert"
)

func getTmpConfig() *node.Config {
	acctAddr := crypto.MustGenerateRandomAddress()

	return &node.Config{
		SeeleCredoConfig: node.SeeleCredoConfig{
			TxConf:   *core.DefaultTxPoolConfig(),
			Coinbase: *acctAddr,
		},
	}
}

func newTestSeeleService() *SeeleCredoService {
	conf := getTmpConfig()
	serviceContext := ServiceContext{
		DataDir: filepath.Join(common.GetTempFolder(), "n1"),
	}

	var key interface{} = "ServiceContext"
	ctx := context.WithValue(context.Background(), key, serviceContext)
	log := log.GetLogger("seeleCredo")

	seeleService, err := NewSeeleCredoService(ctx, conf, log, factory.MustGetConsensusEngine(common.Sha256Algorithm), nil, -1)
	if err != nil {
		panic(err)
	}

	return seeleService
}

func Test_SeeleService_Protocols(t *testing.T) {
	s := newTestSeeleService()
	defer s.Stop()

	protos := s.Protocols()
	assert.Equal(t, len(protos), 1)
}

func Test_SeeleService_Start(t *testing.T) {
	s := newTestSeeleService()
	defer s.Stop()

	s.Start(nil)
	s.Stop()
	assert.Equal(t, s.seeleProtocol == nil, true)
}

func Test_SeeleService_Stop(t *testing.T) {
	s := newTestSeeleService()
	defer s.Stop()

	s.Stop()
	assert.Equal(t, s.chainDB, nil)
	assert.Equal(t, s.accountStateDB, nil)
	assert.Equal(t, s.seeleProtocol == nil, true)

	// can be called more than once
	s.Stop()
	assert.Equal(t, s.chainDB, nil)
	assert.Equal(t, s.accountStateDB, nil)
	assert.Equal(t, s.seeleProtocol == nil, true)
}

func Test_SeeleService_APIs(t *testing.T) {
	s := newTestSeeleService()
	apis := s.APIs()

	assert.Equal(t, len(apis), 10)
	assert.Equal(t, apis[0].Namespace, "seeleCredo")
	assert.Equal(t, apis[1].Namespace, "txpool")
	assert.Equal(t, apis[2].Namespace, "network")
	assert.Equal(t, apis[3].Namespace, "debug")
	assert.Equal(t, apis[4].Namespace, "seeleCredo")
	assert.Equal(t, apis[5].Namespace, "download")
	assert.Equal(t, apis[6].Namespace, "debug")
	assert.Equal(t, apis[7].Namespace, "miner")
	assert.Equal(t, apis[8].Namespace, "txpool")
}
