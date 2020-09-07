/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package common

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/binary"
	"fmt"
	"math/big"
	"regexp"

	"github.com/scdoproject/go-scdo/common/errors"
	"github.com/scdoproject/go-scdo/common/hexutil"
)

//////////////////////////////////////////////////////////////////////////////
// Address format:
// - External account: pubKeyHash[12:32] and set last 4 bits to addressTypeExternal(1)
// - Contract account: AddrNonceHash[14:32] and set last 4 bits to addressTypeContract(2), the left 12 bits for shard (max shard is 4096).
//////////////////////////////////////////////////////////////////////////////

// AddressType represents the address type
type AddressType byte

const (
	// AddressLen length in bytes
	AddressLen = 20

	// AddressTypeExternal is the address type for external account.
	AddressTypeExternal = AddressType(1)

	// AddressTypeContract is the address type for contract account.
	AddressTypeContract = AddressType(2)

	// AddressTypeReserved is the reserved address type for system contract.
	// Note, the address type (4 bits) value ranges [0,15], so the system reserved
	// address type value should greater than 15.
	AddressTypeReserved = AddressType(16)
)

// EmptyAddress presents an empty address
var EmptyAddress = Address{}

// MaxSystemContractAddress max system contract address
var MaxSystemContractAddress = BytesToAddress([]byte{4, 255})

// Address we use public key as node id
type Address [AddressLen]byte

// NewAddress converts a byte slice to a Address
func NewAddress(b []byte) (Address, error) {
	// Validate length
	if len(b) != AddressLen {
		return EmptyAddress, errors.Create(errors.ErrAddressLenInvalid, len(b), AddressLen)
	}

	var id Address
	copy(id[:], b)

	if err := id.Validate(); err != nil {
		return EmptyAddress, err
	}

	return id, nil
}

// PubKeyToAddressOld converts a ECC public key to an external address.
func PubKeyToAddressOld(pubKey *ecdsa.PublicKey, hashFunc func(interface{}) Hash) Address {
	buf := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
	hash := hashFunc(buf[1:]).Bytes()

	var addr Address
	// assume 32 bytes for total key length
	copy(addr[:], hash[32-AddressLen:])

	// modify address type in last byte
	addr[AddressLen-1] &= 0xF0
	addr[AddressLen-1] |= byte(AddressTypeExternal)

	return addr
}

// ValidShard returns true if it is a valid shard number
func ValidShard(shard uint) bool {
	if shard > ShardCount || shard > ShardByte*256-1 || shard == 0 {
		return false
	}
	return true
}

// ValidAccountHex returns true of it is a valid account string
func ValidAccountHex(account string) bool {
	if match, _ := regexp.MatchString("^((1s01|2s02|3s03|4s04|1S01|2S02|3S03|4S04)[a-fA-F0-9]{37}[1-2])|0[sSx]0{40}|0x0[1-4][a-fA-F0-9]{37}[1-2]$", account); !match {
		return false
	}
	return true
}

// PubKeyToAddress converts a ECC public key to an external address.
func PubKeyToAddress(pubKey *ecdsa.PublicKey, shard uint, hashFunc func(interface{}) Hash) (Address, error) {
	if !ValidShard(shard) {
		return EmptyAddress, errors.Create(errors.ErrShardInvalid, shard)
	}
	// Last 20 bytes of public key
	var addr Address
	pubbytes := hashFunc(elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)[1:]).Bytes()
	copy(addr[:], pubbytes[32-AddressLen:])

	// Add shard information in first few bytes
	buf := make([]byte, ShardByte)
	binary.PutUvarint(buf, uint64(shard))
	copy(addr[:ShardByte], buf)

	// Add shard type
	addr[AddressLen-1] &= 0xF0
	addr[AddressLen-1] |= byte(AddressTypeExternal)

	return addr, nil
}

// Validate check whether the address type is valid.
// Two sources of address: external string address, internal publickey conversion
// external Address length enforced in HexToAddress->NewAddress
// internal Address length enforced in LoadECDSAFromString->ToECDSA->toECDSA
func (id *Address) Validate() error {
	if id.IsEmpty() {
		return nil
	}
	if !ValidShard(id.Shard()) {
		return fmt.Errorf("invalid address shard %v", id.Shard())
	}

	if addrType := id.Type(); addrType < AddressTypeReserved && (addrType < AddressTypeExternal || addrType > AddressTypeContract) {
		return fmt.Errorf("invalid address type %v, address = %v", addrType, id.Hex())
	}

	return nil
}

// IsEVMContract indicates whether the address is EVM contract address.
func (id *Address) IsEVMContract() bool {
	return id.Type() == AddressTypeContract
}

// Type returns the address type
func (id *Address) Type() AddressType {
	if id.IsReserved() {
		return AddressTypeReserved
	}

	return AddressType(id[AddressLen-1] & 0x0F)
}

// IsReserved returns true if the address is reserved
func (id *Address) IsReserved() bool {
	return !id.IsEmpty() && bytes.Compare(id.Bytes(), MaxSystemContractAddress.Bytes()) <= 0
}

// Bytes get the actual bytes
//
// Note: if we want to use pointer type, need to change the code snippet in unit test:
//   BytesToAddress([]byte{1, 2}).Bytes()
//   ->
//   addrBytes := BytesToAddress([]byte{1, 2})
//   (&addrBytes).Bytes()
//
// refer link: https://stackoverflow.com/questions/10535743/address-of-a-temporary-in-go
func (id *Address) Bytes() []byte {
	return id[:]
}

// String implements the fmt.Stringer interface
func (id Address) String() string {
	return id.Hex()
}

// Hex converts address to S account string.
func (id Address) Hex() string {

	s := fmt.Sprint(id.Shard())
	a := s + "S" + hexutil.BytesToHex(id.Bytes())[2:]

	return a
}

// Equal checks if this address is the same with the specified address b.
func (id *Address) Equal(b Address) bool {
	return bytes.Equal(id[:], b[:])
}

// IsEmpty returns true if this address is empty. Otherwise, false.
func (id *Address) IsEmpty() bool {
	return id.Equal(EmptyAddress)
}

// HexToAddress converts the specified HEX string to address.
func HexToAddress(id string) (Address, error) {
	if !ValidAccountHex(id) {
		return Address{}, errors.Create(errors.ErrAccountInvalid, id)
	}

	byte, err := hexutil.HexToBytes("0x" + id[2:])
	if err != nil {
		return Address{}, err
	}

	nid, err := NewAddress(byte)
	if err != nil {
		return Address{}, err
	}

	return nid, nil
}

// HexMustToAddres converts the specified HEX string to address.
// Panics on any error.
func HexMustToAddres(id string) Address {
	a, err := HexToAddress(id)
	if err != nil {
		panic(err)
	}

	return a
}

// BytesToAddress converts the specified byte array to Address.
func BytesToAddress(bs []byte) Address {
	var addr Address

	if len(bs) > len(addr) {
		bs = bs[len(bs)-len(addr):]
	}

	copy(addr[len(addr)-len(bs):], bs)

	return addr
}

// BigToAddress converts a big int to address.
func BigToAddress(b *big.Int) Address { return BytesToAddress(b.Bytes()) }

// Big converts address to a big int.
func (id Address) Big() *big.Int { return new(big.Int).SetBytes(id[:]) }

// MarshalText marshals the address to byte array
func (id Address) MarshalText() ([]byte, error) {
	// fmt.Println("marshal", id.Hex())

	str := "0x" + id.Hex()[2:]
	// return []byte(id), nil
	arr := []byte(str)
	return arr, nil
}

// UnmarshalText unmarshals address from HEX string.
func (id *Address) UnmarshalText(json []byte) error {
	a, err := HexToAddress(string(json))
	if err != nil {
		return err
	}

	copy(id[:], a[:])
	return nil
}

// ShardOld returns the shard number of this address.
func (id *Address) ShardOld() uint {
	var sum uint

	// sum [0:18]
	for _, b := range id[:AddressLen-2] {
		sum += uint(b)
	}

	// sum [18:20] except address type
	tail := uint(binary.BigEndian.Uint16(id[AddressLen-2:]))
	sum += (tail >> 4)

	return (sum % ShardCount) + 1
}

// Shard returns shard number of this address
func (id *Address) Shard() uint {
	if id.IsEmpty() {
		return uint(0)
	}
	shard, _ := binary.Uvarint(id[:ShardByte])
	// fmt.Println("SHARD!?", shard, num)
	return uint(shard)
}

// CreateContractAddress returns a contract address that in the same shard of this address.
func (id *Address) CreateContractAddress(nonce uint64, hashFunc func(interface{}) Hash) Address {
	hash := hashFunc([]interface{}{id, nonce})
	return id.CreateContractAddressWithHash(hash)
}

// CreateContractAddressWithHash returns a contract address that in the same shard of this address.
func (id *Address) CreateContractAddressWithHash(h Hash) Address {
	hash := h.Bytes()

	targetShardNum := id.Shard()
	var sum uint

	// sum [14:] of public key hash
	for _, b := range hash[14:] {
		sum += uint(b)
	}

	// sum [18:20] for shard mod and contract address type
	shardNum := (sum % ShardCount) + 1
	encoded := make([]byte, 2)
	var mod uint
	if shardNum <= targetShardNum {
		mod = targetShardNum - shardNum
	} else {
		mod = ShardCount + targetShardNum - shardNum
	}
	mod <<= 4
	mod |= uint(AddressTypeContract) // set address type in the last 4 bits
	binary.BigEndian.PutUint16(encoded, uint16(mod))

	var contractAddr Address
	copy(contractAddr[:AddressLen-2], hash[14:]) // use last 18 bytes of hash (from address + nonce)
	copy(contractAddr[AddressLen-2:], encoded)   // last 2 bytes for shard mod and address type

	buf := make([]byte, ShardByte)
	binary.PutUvarint(buf, uint64(targetShardNum))
	copy(contractAddr[:ShardByte], buf)
	return contractAddr
}
