/**
*  @file
*  @copyright defined in slc/LICENSE
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

	"github.com/seelecredoteam/go-seelecredo/common"
	"github.com/seelecredoteam/go-seelecredo/consensus/factory"
	"github.com/seelecredoteam/go-seelecredo/core"
	"github.com/seelecredoteam/go-seelecredo/crypto"
	"github.com/seelecredoteam/go-seelecredo/log"
	"github.com/seelecredoteam/go-seelecredo/node"
	"github.com/seelecredoteam/go-seelecredo/p2p"
	"github.com/seelecredoteam/go-seelecredo/seeleCredo"
)

func getTmpConfig() *node.Config {
	acctAddr := crypto.MustGenerateShardAddress(1)

	return &node.Config{
		SeeleCredoConfig: node.SeeleCredoConfig{
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
		SeeleCredoConfig: conf.SeeleCredoConfig,
	}

	serviceContext := seeleCredo.ServiceContext{
		DataDir: filepath.Join(common.GetTempFolder(), "n1", fmt.Sprintf("%d", time.Now().UnixNano())),
	}

	ctx := context.WithValue(context.Background(), "ServiceContext", serviceContext)
	dataDir := ctx.Value("ServiceContext").(seeleCredo.ServiceContext).DataDir
	log := log.GetLogger("seeleCredo")

	slcNode, err := node.New(&testConf)
	if err != nil {
		t.Fatal(err)
		return
	}

	slcService, err := seeleCredo.NewSeeleCredoService(ctx, conf, log, factory.MustGetConsensusEngine(common.Sha256Algorithm), nil, -1)
	if err != nil {
		t.Fatal(err)
		return
	}

	monitorService, _ := NewMonitorService(slcService, slcNode, &testConf, log, "run test")

	slcNode.Register(monitorService)
	slcNode.Register(slcService)

	api = NewPublicMonitorAPI(monitorService)

	err = slcNode.Start()
	if err != nil {
		t.Fatal(err)
		return
	}

	slcService.Miner().Start()

	return api, func() {
		api.s.seeleCredo.Stop()
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
			SeeleCredoConfig: conf.SeeleCredoConfig,
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
			SeeleCredoConfig: conf.SeeleCredoConfig,
		}
	}

	serviceContext := seeleCredo.ServiceContext{
		DataDir: common.GetTempFolder() + "/n2/",
	}

	ctx := context.WithValue(context.Background(), "ServiceContext", serviceContext)
	dataDir := ctx.Value("ServiceContext").(seeleCredo.ServiceContext).DataDir
	log := log.GetLogger("seeleCredo")

	slcNode, err := node.New(&testConf)
	if err != nil {
		fmt.Println(err)
		return
	}

	slcService, err := seeleCredo.NewSeeleCredoService(ctx, conf, log, factory.MustGetConsensusEngine(common.Sha256Algorithm), nil, -1)
	if err != nil {
		fmt.Println(err)
		return
	}

	monitorService, _ := NewMonitorService(slcService, slcNode, &testConf, log, "run test")

	slcNode.Register(monitorService)
	slcNode.Register(slcService)

	api = NewPublicMonitorAPI(monitorService)

	if errBranch != 1 {
		slcNode.Start()
	} else {
		slcService.Miner().Start()
	}

	return api, func() {
		api.s.seeleCredo.Stop()
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
