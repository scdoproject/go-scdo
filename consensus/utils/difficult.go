/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package utils

import (
	"math/big"

	"github.com/scdoproject/go-scdo/common"
	"github.com/scdoproject/go-scdo/consensus"
	"github.com/scdoproject/go-scdo/core/types"
)

// getDifficult adjust difficult by parent info
func GetDifficult(time uint64, parentHeader *types.BlockHeader) *big.Int {
	// algorithm:
	// diff = parentDiff + parentDiff / 1024 * max (1 - (blockTime - parentTime) / 20, -99)
	// target block time is 20 seconds
	parentDifficult := parentHeader.Difficulty
	parentTime := parentHeader.CreateTimestamp.Uint64()
	if parentHeader.Height == 0 {
		return parentDifficult
	}

	big1 := big.NewInt(1)
	big99 := big.NewInt(-99)
	big1024 := big.NewInt(1024)
	big2048 := big.NewInt(2048)

	interval := (time - parentTime) / 20
	var x *big.Int
	x = big.NewInt(int64(interval))
	x.Sub(big1, x)
	if x.Cmp(big99) < 0 {
		x = big99
	}

	var y = new(big.Int).Set(parentDifficult)
	if parentHeader.Height < common.SecondForkHeight {
		y.Div(parentDifficult, big2048)
	} else {
		y.Div(parentDifficult, big1024)
	}

	var result = big.NewInt(0)
	result.Mul(x, y)
	result.Add(parentDifficult, result)

	return result
}

// VerifyDifficulty verify the difficulty of the given block based on parent info
func VerifyDifficulty(parent *types.BlockHeader, header *types.BlockHeader) error {
	difficult := GetDifficult(header.CreateTimestamp.Uint64(), parent)
	if difficult.Cmp(header.Difficulty) != 0 {
		return consensus.ErrBlockDifficultInvalid
	}

	return nil
}
