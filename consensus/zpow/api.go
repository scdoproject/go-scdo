/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package zpow

type API struct {
	engine *ZpowEngine
}

// GetDetrate returns the current detrate for local CPU miner and remote miner.
func (api *API) GetDetrate() uint64 {
	return uint64(api.engine.detrate.Rate1())
}

// GetThreads returns the thread number of the miner engine
func (api *API) GetThreads() int {
	return api.engine.threads
}
