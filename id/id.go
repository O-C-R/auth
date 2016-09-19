package id

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
)

var (
	InvalidIDError = errors.New("invalid ID")
)

// ID is a unique identifier.
type ID [20]byte

// New returns a random ID value.
func New() (ID, error) {
	id := ID{}
	if _, err := rand.Read(id[:]); err != nil {
		return id, err
	}

	return id, nil
}

// String returns a hex-encoded string.
func (id ID) String() string {
	return hex.EncodeToString(id[:])
}

// MarshalText returns a hex-encoded slice of bytes.
func (id ID) MarshalText() (text []byte, err error) {
	data := make([]byte, hex.EncodedLen(len(id)))
	hex.Encode(data, id[:])
	return data, nil
}

// UnmarshalText sets the value of the ID based on a hex-encoded slice of bytes.
func (id *ID) UnmarshalText(text []byte) error {
	data := make([]byte, hex.DecodedLen(len(text)))
	n, err := hex.Decode(data, text)
	if err != nil {
		return err
	}

	if n != len(id) {
		return InvalidIDError
	}

	copy(id[:], data)
	return nil
}

// MarshalBinary returns a slice of bytes.
func (id ID) MarshalBinary() ([]byte, error) {
	return id[:], nil
}

// UnmarshalText sets the value of the ID based on a slice of bytes.
func (id *ID) UnmarshalBinary(data []byte) error {
	if len(data) != len(id) {
		return InvalidIDError
	}

	copy(id[:], data)
	return nil
}

// Scan sets the value of the ID based on an interface.
func (id *ID) Scan(src interface{}) error {
	data, ok := src.([]byte)
	if !ok {
		return InvalidIDError
	}

	return id.UnmarshalBinary(data)
}
