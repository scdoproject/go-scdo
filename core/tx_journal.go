package core

import (
	"io"
	"os"

	"github.com/quorum/rlp"
	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/common/errors"
	"github.com/scdoproject/go-scdo/core/types"
	"github.com/scdoproject/mpool/log"
)

// errNoActiveBackup is returned if a transaction is attempted to be inserted
// into the backup, but no such file is currently open.
var errNoActiveBackup = errors.New("no active backup")

// devNil is a WriteCloser that just discards anything written into it. Its
// goal is to allow the transaction backup to write into a fake backup when
// loading transactions on startup without printing warnings due to no file
// being read for write.
type devNil struct{}

func (*devNil) Write(p []byte) (n int, err error) { return len(p), nil }
func (*devNil) Close() error                      { return nil }

type txBackup struct {
	path   string         // file location to store the txs
	writer io.WriteCloser // output stream to write new txs into
}

// newTxBackup create a new tx backup
func newTxBackup(path string) *txBackup {
	return &txBackup{
		path: path,
	}
}

// load parses a tx backup dump from disk, loading its contents to specific pool
func (backup *txBackup) load(add func([]*types.Transaction) []error) error {
	// Skip the parsing if the backup file doesn't exist at all
	if _, err := os.Stat(backup.path); os.IsNotExist(err) {
		return nil
	}
	// Open the backup for loading any past transactions
	input, err := os.Open(backup.path)
	if err != nil {
		return err
	}
	defer input.Close()

	// Temporarily discard any backup additions (don't double add on load)
	backup.writer = new(devNil)
	defer func() { backup.writer = nil }()

	// Inject all transactions from the backup into the pool
	stream := rlp.NewStream(input, 0)
	total, dropped := 0, 0

	// Create a method to load a limited batch of transactions and bump the
	// appropriate progress counters. Then use this method to load all the
	// backuped transactions in small-ish batches.
	loadBatch := func(txs []*types.Transaction) {
		for _, err := range add(txs) {
			if err != nil {
				log.Debug("Failed to add backuped transaction", "err", err)
				dropped++
			}
		}
	}
	var (
		failure error
		batch   []*types.Transaction
	)
	for {
		// Parse the next transaction and terminate on error
		tx := new(types.Transaction)
		if err = stream.Decode(tx); err != nil {
			if err != io.EOF {
				failure = err
			}
			if len(batch) > 0 {
				loadBatch(batch)
			}
			break
		}
		// New transaction parsed, queue up for later, import if threshold is reached
		total++

		if batch = append(batch, tx); len(batch) > 1024 {
			loadBatch(batch)
			batch = batch[:0]
		}
	}
	log.Info("Loaded local transaction backup", "transactions", total, "dropped", dropped)

	return failure
}

// insert adds one specific tx into local disk backup
func (backup *txBackup) insert(tx *types.Transaction) error {
	if backup.writer == nil {
		return errNoActiveBackup
	}
	if err := rlp.Encode(backup.writer, tx); err != nil {
		return err
	}
	return nil
}

// update recreate the tx backup based on current tx poll
func (backup *txBackup) update(all map[common.Address][]types.Transaction) error {
	// need to close current backup if any is open
	if backup.writer != nil {
		if err := backup.writer.Close(); err != nil {
			return err
		}
		backup.writer = nil
	}

	// generate a new backup with the content of the current pool
	replace, err := os.OpenFile(backup.path+"new", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	backuped := 0
	for _, txs := range all {
		for _, tx := range txs {
			if err = rlp.Encode(replace, tx); err != nil {
				replace.Close()
				return err
			}
		}
		backuped += len(txs)
	}
	replace.Close()

	// replace the current backup with the newly generated one
	if err = os.Rename(backup.path+"new", backup.path); err != nil {
		return err
	}
	sink, err := os.OpenFile(backup.path, os.O_WRONLY|os.O_APPEND, 0755)
	if err != nil {
		return err
	}
	backup.writer = sink
	log.Info("Regenerated local transaction backup", "transactions", backuped, "accounts", len(all))

	return nil

}

// close flushed the tx backup content to disk and closed the file
func (backup *txBackup) close() error {
	var err error
	if backup.writer != nil {
		err = backup.writer.Close()
		backup.writer = nil
	}
	return err
}
