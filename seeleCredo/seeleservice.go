/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package seeleCredo

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/seeledevteam/slc/api"
	"github.com/seeledevteam/slc/common"
	"github.com/seeledevteam/slc/consensus"
	"github.com/seeledevteam/slc/core"
	"github.com/seeledevteam/slc/core/store"
	"github.com/seeledevteam/slc/core/types"
	"github.com/seeledevteam/slc/database"
	"github.com/seeledevteam/slc/database/leveldb"
	"github.com/seeledevteam/slc/event"
	"github.com/seeledevteam/slc/log"
	"github.com/seeledevteam/slc/miner"
	"github.com/seeledevteam/slc/node"
	"github.com/seeledevteam/slc/p2p"
	"github.com/seeledevteam/slc/rpc"
	downloader "github.com/seeledevteam/slc/seeleCredo/download"
)

const chainHeaderChangeBuffSize = 100
const maxProgatePeerPerShard = 7

// SeeleCredoService implements full node service.
type SeeleCredoService struct {
	networkID     string
	netVersion    string
	p2pServer     *p2p.Server
	seeleProtocol *SeeleCredoProtocol
	log           *log.SeeleCredoLog

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
func (s *SeeleCredoService) AccountStateDB() database.Database { return s.accountStateDB }

// BlockChain get blockchain
func (s *SeeleCredoService) BlockChain() *core.Blockchain { return s.chain }

// TxPool tx pool
func (s *SeeleCredoService) TxPool() *core.TransactionPool { return s.txPool }

// DebtPool debt pool
func (s *SeeleCredoService) DebtPool() *core.DebtPool { return s.debtPool }

// NetVersion net version
func (s *SeeleCredoService) NetVersion() string { return s.netVersion }

// NetWorkID net id
func (s *SeeleCredoService) NetWorkID() string { return s.networkID }

// Miner get miner
func (s *SeeleCredoService) Miner() *miner.Miner { return s.miner }

// Downloader get downloader
func (s *SeeleCredoService) Downloader() *downloader.Downloader {
	return s.seeleProtocol.Downloader()
}

// P2PServer get p2pServer
func (s *SeeleCredoService) P2PServer() *p2p.Server { return s.p2pServer }

// NewSeeleCredoService create SeeleCredoService
func NewSeeleCredoService(ctx context.Context, conf *node.Config, log *log.SeeleCredoLog, engine consensus.Engine, verifier types.DebtVerifier, startHeight int) (s *SeeleCredoService, err error) {
	s = &SeeleCredoService{
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

	s.miner = miner.NewMiner(conf.SeeleCredoConfig.Coinbase, s, s.debtVerifier, engine)

	// initialize and validate genesis
	if err = s.initGenesisAndChain(&serviceContext, conf, startHeight); err != nil {
		return nil, err
	}

	if err = s.initPool(conf); err != nil {
		return nil, err
	}

	if s.seeleProtocol, err = NewSeeleCredoProtocol(s, log); err != nil {
		s.Stop()
		log.Error("failed to create seeleProtocol in NewSeeleCredoService, %s", err)
		return nil, err
	}

	return s, nil
}

func (s *SeeleCredoService) initBlockchainDB(serviceContext *ServiceContext) (err error) {
	s.chainDBPath = filepath.Join(serviceContext.DataDir, BlockChainDir)
	s.log.Info("NewSeeleCredoService BlockChain datadir is %s", s.chainDBPath)

	if s.chainDB, err = leveldb.NewLevelDB(s.chainDBPath); err != nil {
		s.log.Error("NewSeeleCredoService Create BlockChain err. %s", err)
		return err
	}

	return nil
}

func (s *SeeleCredoService) initAccountStateDB(serviceContext *ServiceContext) (err error) {
	s.accountStateDBPath = filepath.Join(serviceContext.DataDir, AccountStateDir)
	s.log.Info("NewSeeleCredoService account state datadir is %s", s.accountStateDBPath)

	if s.accountStateDB, err = leveldb.NewLevelDB(s.accountStateDBPath); err != nil {
		s.Stop()
		s.log.Error("NewSeeleCredoService Create BlockChain err: failed to create account state DB, %s", err)
		return err
	}

	return nil
}

func (s *SeeleCredoService) initDebtManagerDB(serviceContext *ServiceContext) (err error) {
	s.debtManagerDBPath = filepath.Join(serviceContext.DataDir, DebtManagerDir)
	s.log.Info("NewSeeleCredoService debt manager datadir is %s", s.debtManagerDBPath)

	if s.debtManagerDB, err = leveldb.NewLevelDB(s.debtManagerDBPath); err != nil {
		s.Stop()
		s.log.Error("NewSeeleCredoService Create BlockChain err: failed to create debt manager DB, %s", err)
		return err
	}

	return nil
}

func (s *SeeleCredoService) initGenesisAndChain(serviceContext *ServiceContext, conf *node.Config, startHeight int) (err error) {
	bcStore := store.NewCachedStore(store.NewBlockchainDatabase(s.chainDB))
	genesis := core.GetGenesis(&conf.SeeleCredoConfig.GenesisConfig)

	if err = genesis.InitializeAndValidate(bcStore, s.accountStateDB); err != nil {
		s.Stop()
		s.log.Error("NewSeeleCredoService genesis.Initialize err. %s", err)
		return err
	}

	recoveryPointFile := filepath.Join(serviceContext.DataDir, BlockChainRecoveryPointFile)
	if s.chain, err = core.NewBlockchain(bcStore, s.accountStateDB, recoveryPointFile, s.miner.GetEngine(), s.debtVerifier, startHeight); err != nil {
		s.Stop()
		s.log.Error("failed to init chain in NewSeeleCredoService. %s", err)
		return err
	}

	return nil
}

func (s *SeeleCredoService) initPool(conf *node.Config) (err error) {
	if s.lastHeader, err = s.chain.GetStore().GetHeadBlockHash(); err != nil {
		s.Stop()
		return fmt.Errorf("failed to get chain header, %s", err)
	}

	s.chainHeaderChangeChannel = make(chan common.Hash, chainHeaderChangeBuffSize)
	s.debtPool = core.NewDebtPool(s.chain, s.debtVerifier)
	s.txPool = core.NewTransactionPool(conf.SeeleCredoConfig.TxConf, s.chain)

	event.ChainHeaderChangedEventMananger.AddAsyncListener(s.chainHeaderChanged)
	go s.MonitorChainHeaderChange()

	return nil
}

// chainHeaderChanged handle chain header changed event.
// add forked transaction back
// deleted invalid transaction
func (s *SeeleCredoService) chainHeaderChanged(e event.Event) {
	newBlock := e.(*types.Block)
	if newBlock == nil || newBlock.HeaderHash.IsEmpty() {
		return
	}

	s.chainHeaderChangeChannel <- newBlock.HeaderHash
}

// MonitorChainHeaderChange monitor and handle chain header event
func (s *SeeleCredoService) MonitorChainHeaderChange() {
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
func (s *SeeleCredoService) Protocols() (protos []p2p.Protocol) {
	protos = append(protos, s.seeleProtocol.Protocol)
	return protos
}

// Start implements node.Service, starting goroutines needed by SeeleCredoService.
func (s *SeeleCredoService) Start(srvr *p2p.Server) error {
	s.p2pServer = srvr
	s.seeleProtocol.Start()

	return nil
}

// Stop implements node.Service, terminating all internal goroutines.
func (s *SeeleCredoService) Stop() error {
	//TODO
	// s.txPool.Stop() s.chain.Stop()
	// retries? leave it to future
	if s.seeleProtocol != nil {
		s.seeleProtocol.Stop()
		s.seeleProtocol = nil
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

// APIs implements node.Service, returning the collection of RPC services the seeleCredo package offers.
// must to make sure that the order of the download api is 5; we get the download api by 5
func (s *SeeleCredoService) APIs() (apis []rpc.API) {
	apis = append(apis, api.GetAPIs(NewSeeleCredoBackend(s))...)
	apis = append(apis, []rpc.API{
		{
			Namespace: "seeleCredo",
			Version:   "1.0",
			Service:   NewPublicSeeleAPI(s),
			Public:    true,
		},
		{
			Namespace: "download",
			Version:   "1.0",
			Service:   downloader.NewPrivatedownloaderAPI(s.seeleProtocol.downloader),
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
