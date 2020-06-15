/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package monitor

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/consensus/factory"
	"github.com/scdoproject/go-scdo/core"
	"github.com/scdoproject/go-scdo/crypto"
	"github.com/scdoproject/go-scdo/log"
	"github.com/scdoproject/go-scdo/node"
	"github.com/scdoproject/go-scdo/p2p"
	"github.com/scdoproject/go-scdo/scdo"
)

func getTmpConfig() *node.Config {
	acctAddr := crypto.MustGenerateShardAddress(1)

	return &node.Config{
		ScdoConfig: node.ScdoConfig{
			TxConf:   *core.DefaultTxPoolConfig(),
			Coinbase: *acctAddr,
			GenesisConfig: core.GenesisInfo{
				Difficult:       1,
				ShardNumber:     1,
				CreateTimestamp: big.NewInt(0),
			},
		},
	}
}

func createTestAPI(t *testing.T) (api *PublicMonitorAPI, dispose func()) {
	conf := getTmpConfig()
	key, _ := crypto.GenerateKey()
	testConf := node.Config{
		BasicConfig: node.BasicConfig{
			Name:    "Node for test",
			Version: "Test 1.0",
			DataDir: "node1",
			RPCAddr: "127.0.0.1:8027",
		},
		P2PConfig: p2p.Config{
			PrivateKey: key,
			ListenAddr: "0.0.0.0:8037",
		},
		WSServerConfig: node.WSServerConfig{
			Address:      "127.0.0.1:8047",
			CrossOrigins: []string{"*"},
		},
		ScdoConfig: conf.ScdoConfig,
	}

	serviceContext := scdo.ServiceContext{
		DataDir: filepath.Join(common.GetTempFolder(), "n1", fmt.Sprintf("%d", time.Now().UnixNano())),
	}

	ctx := context.WithValue(context.Background(), "ServiceContext", serviceContext)
	dataDir := ctx.Value("ServiceContext").(scdo.ServiceContext).DataDir
	log := log.GetLogger("scdo")

	scdoNode, err := node.New(&testConf)
	if err != nil {
		t.Fatal(err)
		return
	}

	scdoService, err := scdo.NewScdoService(ctx, conf, log, factory.MustGetConsensusEngine(common.Sha256Algorithm), nil, -1)
	if err != nil {
		t.Fatal(err)
		return
	}

	monitorService, _ := NewMonitorService(scdoService, scdoNode, &testConf, log, "run test")

	scdoNode.Register(monitorService)
	scdoNode.Register(scdoService)

	api = NewPublicMonitorAPI(monitorService)

	err = scdoNode.Start()
	if err != nil {
		t.Fatal(err)
		return
	}

	scdoService.Miner().Start()

	return api, func() {
		api.s.scdo.Stop()
		os.RemoveAll(dataDir)
	}
}

func createTestAPIErr(errBranch int) (api *PublicMonitorAPI, dispose func()) {
	conf := getTmpConfig()

	testConf := node.Config{}
	if errBranch == 1 {

		key, _ := crypto.GenerateKey()
		testConf = node.Config{
			BasicConfig: node.BasicConfig{
				Name:    "Node for test2",
				Version: "Test 1.0",
				DataDir: "node1",
				RPCAddr: "127.0.0.1:55028",
			},
			P2PConfig: p2p.Config{
				PrivateKey: key,
				ListenAddr: "0.0.0.0:39008",
			},
			ScdoConfig: conf.ScdoConfig,
		}
	} else {
		key, _ := crypto.GenerateKey()
		testConf = node.Config{
			BasicConfig: node.BasicConfig{
				Name:    "Node for test3",
				Version: "Test 1.0",
				DataDir: "node1",
				RPCAddr: "127.0.0.1:55029",
			},
			P2PConfig: p2p.Config{
				PrivateKey: key,
				ListenAddr: "0.0.0.0:39009",
			},
			ScdoConfig: conf.ScdoConfig,
		}
	}

	serviceContext := scdo.ServiceContext{
		DataDir: common.GetTempFolder() + "/n2/",
	}

	ctx := context.WithValue(context.Background(), "ServiceContext", serviceContext)
	dataDir := ctx.Value("ServiceContext").(scdo.ServiceContext).DataDir
	log := log.GetLogger("scdo")

	scdoNode, err := node.New(&testConf)
	if err != nil {
		fmt.Println(err)
		return
	}

	scdoService, err := scdo.NewScdoService(ctx, conf, log, factory.MustGetConsensusEngine(common.Sha256Algorithm), nil, -1)
	if err != nil {
		fmt.Println(err)
		return
	}

	monitorService, _ := NewMonitorService(scdoService, scdoNode, &testConf, log, "run test")

	scdoNode.Register(monitorService)
	scdoNode.Register(scdoService)

	api = NewPublicMonitorAPI(monitorService)

	if errBranch != 1 {
		scdoNode.Start()
	} else {
		scdoService.Miner().Start()
	}

	return api, func() {
		api.s.scdo.Stop()
		os.RemoveAll(dataDir)
	}
}

func Test_PublicMonitorAPI_Allright(t *testing.T) {
	api, dispose := createTestAPI(t)
	defer dispose()
	if api == nil {
		t.Fatal("failed to create api")
	}

	_, err := api.NodeInfo()
	if err != nil {
		t.Fatalf("failed to get nodeInfo: %v", err)
	}

	if _, err := api.NodeStats(); err != nil {
		t.Fatalf("failed to get nodeInfo: %v", err)
	}
}

func Test_PublicMonitorAPI_Err(t *testing.T) {
	api, dispose := createTestAPIErr(1)
	defer dispose()
	if api == nil {
		t.Fatal("failed to create api")
	}

	if _, err := api.NodeStats(); err == nil {
		t.Fatalf("error branch is not covered")
	}
}
