/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package errors

// ErrorCode represents the error code type in scdo.
type ErrorCode int

const (
	errCommon ErrorCode = 1000 + iota
	// ErrDecrypt error when the passphrase is not right.
	ErrDecrypt
	// ErrEmptyAuthKey error when the auth key is empty.
	ErrEmptyAuthKey
	// ErrKeyVersionMismatch error when the auth key version does not match.
	ErrKeyVersionMismatch
	// ErrAddressLenInvalid is returned when the address length is invalid.
	ErrAddressLenInvalid
	// ErrPasswordRepeatMismatch is returned when the repeat password is not equal to the origin one.
	ErrPasswordRepeatMismatch
)

const (
	errCore ErrorCode = 2000 + iota
	// @todo define errors under core pkg here
)

var constErrors = map[ErrorCode]error{
	ErrDecrypt:                newScdoError(ErrDecrypt, "could not decrypt key with given passphrase"),
	ErrEmptyAuthKey:           newScdoError(ErrEmptyAuthKey, "encryption auth key could not be empty"),
	ErrPasswordRepeatMismatch: newScdoError(ErrPasswordRepeatMismatch, "repeat password is not equal to orignal one"),
}

var parameterizedErrors = map[ErrorCode]string{
	ErrKeyVersionMismatch: "Version not supported: %v",
	ErrAddressLenInvalid:  "invalid address length %v, expected length is %v",
}
