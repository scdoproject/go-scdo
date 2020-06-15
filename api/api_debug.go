/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package api

// PrivateDebugAPI provides an API to access full node-related information for debugging.
type PrivateDebugAPI struct {
	s Backend
}

// NewPrivateDebugAPI creates a new NewPrivateDebugAPI object for rpc service.
func NewPrivateDebugAPI(s Backend) *PrivateDebugAPI {
	return &PrivateDebugAPI{s}
}
