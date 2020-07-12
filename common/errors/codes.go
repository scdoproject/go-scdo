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
	// ErrShardInvalid is returned when the shard is not valid
	ErrShardInvalid
	// ErrAccountInvalid is returned when the account string is not valid
	ErrAccountInvalid
)

const (
	errCore ErrorCode = 2000 + iota
	// @todo define errors under core pkg here
)

var constErrors = map[ErrorCode]error{
	ErrDecrypt:                newScdoError(ErrDecrypt, "Could not decrypt key with given passphrase"),
	ErrEmptyAuthKey:           newScdoError(ErrEmptyAuthKey, "Encryption auth key could not be empty"),
	ErrPasswordRepeatMismatch: newScdoError(ErrPasswordRepeatMismatch, "Repeat password is not equal to orignal one"),
}

var parameterizedErrors = map[ErrorCode]string{
	ErrKeyVersionMismatch: "Version not supported: %v",
	ErrAddressLenInvalid:  "Invalid address length %v, expected length is %v",
	ErrShardInvalid:       "Shard number invalid: %v",
	ErrAccountInvalid:     "Invalid account string: %v",
}
