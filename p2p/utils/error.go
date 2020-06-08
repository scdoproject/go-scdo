/**
*  @file
*  @copyright defined in slc/LICENSE
 */
package utils

// IsTemporaryError checks whether the given error should be considered temporary.
func IsTemporaryError(err error) bool {
	tempErr, ok := err.(interface {
		Temporary() bool
	})
	return ok && tempErr.Temporary()
}
