// Package types
package types

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"log"

	"github.com/golang/glog"
)

const (
	// ServiceHash is a random number to be included in all the signs to enforce the hashes creation
	ServiceHash = "1d0684170dcf58ed2499d233be72b5dde48d8124cb617f1309bae85da2fe85cf"
	// BlockHashPrefix is the standard prefix used in DarkMatter protocol to recognize their blocks hashes
	BlockHashPrefix = "dd"
)

// KVStore defines a KV pair storage manager definition
type KVStore interface {
	StoreValue(key string, value []byte) error // StoreValue saves a unespecified instance using an string as key
	GetValue(key string) ([]byte, error)
	StoreBlock(block FullSignedBlock) error
	GetBlock(hash string) (*FullSignedBlock, error)
	GetLatestBlocks(timestamp uint64, n int) ([]FullSignedBlock, error)
	FindBlockByTimestamp(timestamp uint64) (*FullSignedBlock, error)
	FindBlockByHeight(Height uint64) (*FullSignedBlock, error)
}

// FullSignedBlock is the message to send to the connected clients through websocket
type FullSignedBlock struct {
	Hash            string      `json:"hash"`
	Height          uint64      `json:"height"`
	Timestamp       uint64      `json:"timestamp"`
	Payload         interface{} `json:"payload"`
	PreviousHash    string      `json:"previousHash"`
	Address         string      `json:"address"`
	PreviousAddress string      `json:"previousAddress"`
	Memo            string      `json:"memo"`
	Evidence        interface{} `json:"evidence"`
}

// CreateHash calculates the hash for a block
func (block *FullSignedBlock) CreateHash() error {

	// create a hash the result
	block.Hash = "" // To asure a clean hash
	hash, err := CalculateHash(block)
	if err == nil {
		block.Hash = hash
	}

	// The hashes for the block has attached a prefix and the the number of seconds taken from the timestamp
	seconds := time.Unix(int64(block.Timestamp), 0).Second()
	block.Hash = fmt.Sprintf("%s%02d%s", BlockHashPrefix, seconds, block.Hash)

	return err // No error
}

// string implement the Stringer interface
func (block FullSignedBlock) String() string {
	bytes, err := json.Marshal(block)
	if err != nil {
		log.Fatal("Error deserializing a block", err)
	}

	return string(bytes)
}

// CalculateHash generate a hash using a double operation over the serialized content of object
func CalculateHash(obj interface{}) (string, error) {
	bytes, err := json.Marshal(obj)
	if err != nil {
		glog.Errorf("Error serializing message", err)
		return "", err
	}
	// Sign the content of block including the hash of DarkMatter
	rawContent := fmt.Sprintf("%s:%s", ServiceHash, bytes)

	// Double hash for the content
	doubleHash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawContent)))
	doubleHash = fmt.Sprintf("%x", sha256.Sum256([]byte(doubleHash)))

	return doubleHash, nil
}
