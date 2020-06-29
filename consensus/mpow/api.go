/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package mpow

type API struct {
	engine *MpowEngine
}

// GetThreads returns the thread number of the miner engine
func (api *API) GetThreads() int {
	return api.engine.threads
}
