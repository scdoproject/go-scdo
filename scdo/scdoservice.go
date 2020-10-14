/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package scdo

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/scdoproject/go-scdo/api"
	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/consensus"
	"github.com/scdoproject/go-scdo/core"
	"github.com/scdoproject/go-scdo/core/store"
	"github.com/scdoproject/go-scdo/core/types"
	"github.com/scdoproject/go-scdo/database"
	"github.com/scdoproject/go-scdo/database/leveldb"
	"github.com/scdoproject/go-scdo/event"
	"github.com/scdoproject/go-scdo/log"
	"github.com/scdoproject/go-scdo/miner"
	"github.com/scdoproject/go-scdo/node"
	"github.com/scdoproject/go-scdo/p2p"
	"github.com/scdoproject/go-scdo/p2p/discovery"
	"github.com/scdoproject/go-scdo/rpc"
	downloader "github.com/scdoproject/go-scdo/scdo/download"
)

const chainHeaderChangeBuffSize = 100
const maxProgatePeerPerShard = 7

// ScdoService implements full node service.
type ScdoService struct {
	networkID    string
	netVersion   string
	p2pServer    *p2p.Server
	udpServer    *discovery.UDP
	scdoProtocol *ScdoProtocol
	log          *log.ScdoLog

	txPool             *core.TransactionPool
	debtPool           *core.DebtPool
	chain              *core.Blockchain
	chainDB            database.Database // database used to store blocks.
	chainDBPath        string
	accountStateDB     database.Database // database used to store account state info.
	accountStateDBPath string
	debtManagerDB      database.Database // database used to store debts in debt manager.
	debtManagerDBPath  string
	miner              *miner.Miner

	lastHeader               common.Hash
	chainHeaderChangeChannel chan common.Hash

	debtVerifier types.DebtVerifier
}

// ServiceContext is a collection of service configuration inherited from node
type ServiceContext struct {
	DataDir string
}

// AccountStateDB return account state db
func (s *ScdoService) AccountStateDB() database.Database { return s.accountStateDB }

// BlockChain get blockchain
func (s *ScdoService) BlockChain() *core.Blockchain { return s.chain }

// TxPool tx pool
func (s *ScdoService) TxPool() *core.TransactionPool { return s.txPool }

// DebtPool debt pool
func (s *ScdoService) DebtPool() *core.DebtPool { return s.debtPool }

// NetVersion net version
func (s *ScdoService) NetVersion() string { return s.netVersion }

// NetWorkID net id
func (s *ScdoService) NetWorkID() string { return s.networkID }

// Miner get miner
func (s *ScdoService) Miner() *miner.Miner { return s.miner }

// Downloader get downloader
func (s *ScdoService) Downloader() *downloader.Downloader {
	return s.scdoProtocol.Downloader()
}

// P2PServer get p2pServer
func (s *ScdoService) P2PServer() *p2p.Server { return s.p2pServer }

// NewScdoService create ScdoService
func NewScdoService(ctx context.Context, conf *node.Config, log *log.ScdoLog, engine consensus.Engine, verifier types.DebtVerifier, startHeight int, isPoolMode bool) (s *ScdoService, err error) {
	s = &ScdoService{
		log:          log,
		networkID:    conf.P2PConfig.NetworkID,
		netVersion:   conf.BasicConfig.Version,
		debtVerifier: verifier,
	}

	serviceContext := ctx.Value("ServiceContext").(ServiceContext)

	// Initialize blockchain DB.
	if err = s.initBlockchainDB(&serviceContext); err != nil {
		return nil, err
	}

	leveldb.StartMetrics(s.chainDB, "chaindb", log)

	// Initialize account state info DB.
	if err = s.initAccountStateDB(&serviceContext); err != nil {
		return nil, err
	}

	// Initialize debt manager DB.
	if err = s.initDebtManagerDB(&serviceContext); err != nil {
		return nil, err
	}

	s.miner = miner.NewMiner(conf.ScdoConfig.Coinbase, conf.ScdoConfig.CoinbaseList, s, s.debtVerifier, engine, isPoolMode)

	// initialize and validate genesis
	if err = s.initGenesisAndChain(&serviceContext, conf, startHeight); err != nil {
		return nil, err
	}

	if err = s.initPool(conf); err != nil {
		return nil, err
	}

	if s.scdoProtocol, err = NewScdoProtocol(s, log); err != nil {
		s.Stop()
		log.Error("failed to create scdoProtocol in NewScdoService, %s", err)
		return nil, err
	}

	return s, nil
}

func (s *ScdoService) initBlockchainDB(serviceContext *ServiceContext) (err error) {
	s.chainDBPath = filepath.Join(serviceContext.DataDir, BlockChainDir)
	s.log.Info("NewScdoService BlockChain datadir is %s", s.chainDBPath)

	if s.chainDB, err = leveldb.NewLevelDB(s.chainDBPath); err != nil {
		s.log.Error("NewScdoService Create BlockChain err. %s", err)
		return err
	}

	return nil
}

func (s *ScdoService) initAccountStateDB(serviceContext *ServiceContext) (err error) {
	s.accountStateDBPath = filepath.Join(serviceContext.DataDir, AccountStateDir)
	s.log.Info("NewScdoService account state datadir is %s", s.accountStateDBPath)

	if s.accountStateDB, err = leveldb.NewLevelDB(s.accountStateDBPath); err != nil {
		s.Stop()
		s.log.Error("NewScdoService Create BlockChain err: failed to create account state DB, %s", err)
		return err
	}

	return nil
}

func (s *ScdoService) initDebtManagerDB(serviceContext *ServiceContext) (err error) {
	s.debtManagerDBPath = filepath.Join(serviceContext.DataDir, DebtManagerDir)
	s.log.Info("NewScdoService debt manager datadir is %s", s.debtManagerDBPath)

	if s.debtManagerDB, err = leveldb.NewLevelDB(s.debtManagerDBPath); err != nil {
		s.Stop()
		s.log.Error("NewScdoService Create BlockChain err: failed to create debt manager DB, %s", err)
		return err
	}

	return nil
}

func (s *ScdoService) initGenesisAndChain(serviceContext *ServiceContext, conf *node.Config, startHeight int) (err error) {
	bcStore := store.NewCachedStore(store.NewBlockchainDatabase(s.chainDB))
	genesis := core.GetGenesis(&conf.ScdoConfig.GenesisConfig)

	if err = genesis.InitializeAndValidate(bcStore, s.accountStateDB); err != nil {
		s.Stop()
		s.log.Error("NewScdoService genesis.Initialize err. %s", err)
		return err
	}

	recoveryPointFile := filepath.Join(serviceContext.DataDir, BlockChainRecoveryPointFile)
	if s.chain, err = core.NewBlockchain(bcStore, s.accountStateDB, recoveryPointFile, s.miner.GetEngine(), s.debtVerifier, startHeight); err != nil {
		s.Stop()
		s.log.Error("failed to init chain in NewScdoService. %s", err)
		return err
	}

	return nil
}

func (s *ScdoService) initPool(conf *node.Config) (err error) {
	if s.lastHeader, err = s.chain.GetStore().GetHeadBlockHash(); err != nil {
		s.Stop()
		return fmt.Errorf("failed to get chain header, %s", err)
	}

	s.chainHeaderChangeChannel = make(chan common.Hash, chainHeaderChangeBuffSize)
	s.debtPool = core.NewDebtPool(s.chain, s.debtVerifier)
	s.txPool = core.NewTransactionPool(conf.ScdoConfig.TxConf, s.chain)

	event.ChainHeaderChangedEventMananger.AddAsyncListener(s.chainHeaderChanged)
	go s.MonitorChainHeaderChange()

	return nil
}

// chainHeaderChanged handle chain header changed event.
// add forked transaction back
// deleted invalid transaction
func (s *ScdoService) chainHeaderChanged(e event.Event) {
	newBlock := e.(*types.Block)
	if newBlock == nil || newBlock.HeaderHash.IsEmpty() {
		return
	}

	s.chainHeaderChangeChannel <- newBlock.HeaderHash
}

// MonitorChainHeaderChange monitor and handle chain header event
func (s *ScdoService) MonitorChainHeaderChange() {
	for {
		select {
		case newHeader := <-s.chainHeaderChangeChannel:
			if s.lastHeader.IsEmpty() {
				s.lastHeader = newHeader
				return
			}

			s.txPool.HandleChainHeaderChanged(newHeader, s.lastHeader)
			s.debtPool.HandleChainHeaderChanged(newHeader, s.lastHeader)

			s.lastHeader = newHeader
		}
	}
}

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *ScdoService) Protocols() (protos []p2p.Protocol) {
	protos = append(protos, s.scdoProtocol.Protocol)
	return protos
}

// Start implements node.Service, starting goroutines needed by ScdoService.
func (s *ScdoService) Start(srvr *p2p.Server) error {
	s.p2pServer = srvr
	s.scdoProtocol.Start()

	return nil
}

// Stop implements node.Service, terminating all internal goroutines.
func (s *ScdoService) Stop() error {
	//TODO
	// s.txPool.Stop() s.chain.Stop()
	// retries? leave it to future
	if s.scdoProtocol != nil {
		s.scdoProtocol.Stop()
		s.scdoProtocol = nil
	}

	if s.chainDB != nil {
		s.chainDB.Close()
		s.chainDB = nil
	}

	if s.accountStateDB != nil {
		s.accountStateDB.Close()
		s.accountStateDB = nil
	}

	if s.debtManagerDB != nil {
		s.debtManagerDB.Close()
		s.debtManagerDB = nil
	}

	return nil
}

// APIs implements node.Service, returning the collection of RPC services the scdo package offers.
// must to make sure that the order of the download api is 5; we get the download api by 5
func (s *ScdoService) APIs() (apis []rpc.API) {
	apis = append(apis, api.GetAPIs(NewScdoBackend(s))...)
	apis = append(apis, []rpc.API{
		{
			Namespace: "scdo",
			Version:   "1.0",
			Service:   NewPublicSeeleAPI(s),
			Public:    true,
		},
		{
			Namespace: "download",
			Version:   "1.0",
			Service:   downloader.NewPrivatedownloaderAPI(s.scdoProtocol.downloader),
			Public:    true,
		},
		{
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s),
			Public:    false,
		},
		{
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		},
		{
			Namespace: "txpool",
			Version:   "1.0",
			Service:   NewTransactionPoolAPI(s),
			Public:    true,
		},
	}...)

	minerApis := s.miner.GetEngine().APIs(s.chain)
	apis = append(apis, minerApis...)

	return apis
}
