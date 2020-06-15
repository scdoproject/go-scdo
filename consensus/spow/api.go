/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

 package spow

 type API struct {
	 engine *SpowEngine
 }
 
 // GetThreads returns the thread number of the miner engine
 func (api *API) GetThreads() int {
	 return api.engine.threads
 }
