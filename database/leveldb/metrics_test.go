/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package leveldb

import (
	"os"
	"testing"
	"time"

	"github.com/scdoproject/go-scdo/log"
)

func Test_StartMetrics(t *testing.T) {
	// Init LevelDB
	dir := prepareDbFolder("", "leveldbtest")
	defer os.RemoveAll(dir)
	db := newDbInstance(dir)
	defer db.Close()

	StartMetrics(db, "testdb", log.GetLogger("test"))

	select {
	case <-time.After(2 * time.Second):
	}

	if lvdb, ok := db.(*LevelDB); ok {
		if lvdb.quitChan == nil {
			t.Fatalf("error in collect metrics")
		}
	} else {
		t.Fatalf("db type is not 'LevelDB'")
	}
}
