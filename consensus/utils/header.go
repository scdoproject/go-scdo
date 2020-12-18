/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package utils

import (
	"github.com/scdoproject/go-scdo/consensus"
	"github.com/scdoproject/go-scdo/core/types"
)

// VerifyHeaderCommon verify the height, timestamp and difficulty
// of the given header based on parent info
func VerifyHeaderCommon(header, parent *types.BlockHeader) error {
	if header.Height != parent.Height+1 {
		return consensus.ErrBlockInvalidHeight
	}

	if header.CreateTimestamp.Cmp(parent.CreateTimestamp) < 0 {
		return consensus.ErrBlockCreateTimeOld
	}

	if err := VerifyDifficulty(parent, header); err != nil {
		return err
	}

	return nil
}
