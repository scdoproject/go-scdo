/**
* @file
* @copyright defined in scdo/LICENSE
 */

package core

import "time"

// TransactionPoolConfig is the configuration of the transaction pool.
type TransactionPoolConfig struct {
	Capacity                int           // Maximum number of transactions in the pool.
	PriceLimit              uint64        // Minimum gas price to enforce for acceptance into the pool
	PriceBump               uint64        // Minimum price bump percentage to replace an already existing transaction (nonce)
	Backup                  string        // backup of local transactions to survive node restarts
	ToBackup                time.Duration // Time interval to regenerate the local transaction backup
	DebtPoolCapacity        int
	ToConfirmedDebtCapacity int
	// DebtManagerPoolCapacity int
}

// DefaultTxPoolConfig returns the default configuration of the transaction pool.
func DefaultTxPoolConfig() *TransactionPoolConfig {
	return &TransactionPoolConfig{
		// 1 simple transaction is about 152 byte size. So 1000 transactions is about 152KB, and 10000 transaction is about 1.52MB.
		// We want to cache transactions for about 100 blocks (about 500k transactions), which means at least 25 minutes block generation consume,
		// the memory usage will be <=100MB for tx pool.
		// in real test. 100000 transaction will use 100MB memory. so we will set capacity to 200000, which is about 200MB memory usage.
		Capacity:                200000,
		PriceLimit:              1,
		PriceBump:               10,
		DebtPoolCapacity:        100000, // DebtPoolCapacity we need bigger capacity to hold more debt in real test. the memory usage for 100000 will be about 150MB
		ToConfirmedDebtCapacity: 100000, // ToConfirmedDebtCapacity capacity for to confirmed debt map in real test. the memory usage for 100000 will be about 150MB
		// DebtManagerPoolCapacity: 100000, // DebtManagerPoolCapacity capacity for to-be-sent debt in real test. the memory usage for 100000 will be about 150MB
	}
}

// // DebtPoolCapacity we need bigger capacity to hold more debt in real test. the memory usage for 100000 will be about 150MB
// var DebtPoolCapacity = 100000

// var DebtPoolPriceBump = uint64(10)

// // ToConfirmedDebtCapacity capacity for to confirmed debt map in real test. the memory usage for 100000 will be about 150MB
// var ToConfirmedDebtCapacity = 100000

// // DebtManagerPoolCapacity capacity for to-be-sent debt in real test. the memory usage for 100000 will be about 150MB
var DebtManagerPoolCapacity = 100000
