package database

import (
	"encoding/binary"
	"encoding/json"
	"unsafe"

	"github.com/golang/glog"

	"github.com/aquarelle-tech/darkmatter/types"
	"github.com/dgraph-io/badger"
)

const (
	// Prefixes indentify each key in the datastore
	HashKeyPrefix      = 0x1
	TimestampKeyPrefix = 0x2
	HeightKeyPrefix    = 0x3
	FixedKeyPrefix     = 0xFF // Any other key
)

// Store implements the KVStore interface
type Store struct {
	StorFileLocation string
	storHandler      *badger.DB
}

// NewKVStore creates a new store for key-value pairs
func NewKVStore(locationDirectory string) types.KVStore {

	// Open badger
	options := badger.DefaultOptions(locationDirectory)
	options.Truncate = true // To avoid problems with Windows. WARNING

	stor, err := badger.Open(options)
	if err != nil {
		panic(err)
	}

	kvs := &Store{
		StorFileLocation: locationDirectory,
		storHandler:      stor,
	}

	return kvs
}

// storeUIntIndex store a value in the database indexed by an uint64
func storeUIntIndex(txn *badger.Txn, key uint64, value []byte, prefix byte) error {

	index := make([]byte, 8)
	binary.BigEndian.PutUint64(index, key)
	index = append([]byte{prefix}, index...)

	return txn.Set(index, value)
}

// readUIntIndex read a value from the database indexed by an uint64
func readUIntIndex(txn *badger.Txn, key uint64, prefix byte) ([]byte, error) {

	index := make([]byte, 8)
	binary.BigEndian.PutUint64(index, key)
	index = append([]byte{prefix}, index...)

	item, err := txn.Get(index)
	if err != nil {
		return nil, err
	}

	return item.ValueCopy(nil)
}

// storeStringIndex store a value in the database indexed by an uint64
func storeStringIndex(txn *badger.Txn, key string, value []byte, prefix byte) error {

	index := append([]byte{prefix}, []byte(key)...)
	return txn.Set(index, value)
}

// readStringIndex read a value from the database indexed by an uint64
func readStringIndex(txn *badger.Txn, key string, prefix byte) ([]byte, error) {

	index := append([]byte{prefix}, []byte(key)...)
	item, err := txn.Get(index)
	if err != nil {
		return nil, err
	}

	return item.ValueCopy(nil)
}

// StoreBlock store a full block in the database. The block will be indexed by their timestamp and Height
func (s Store) StoreBlock(block types.FullSignedBlock) error {

	glog.Infof("Storing block %s", block.Hash)

	// Serialize all the parts: block in json
	bytes, err := json.Marshal(block)
	glog.Infof("Marshaled content: %d bytes (payload length: %d bytes)", len(bytes), unsafe.Sizeof(block.Payload))

	err = s.storHandler.Update(func(txn *badger.Txn) error {

		var txErr error
		// Store the hash as a key. This is the main register
		if txErr = storeStringIndex(txn, block.Hash, bytes, HashKeyPrefix); txErr == nil {
			glog.Infof("Stored block %s - Height = %d, Payload len = %d", block.Hash, block.Height, unsafe.Sizeof(block.Payload))
			// And now store the indexes. Using this indexes it is possible to retrieve the hash, and next the block
			if txErr = storeUIntIndex(txn, block.Timestamp, []byte(block.Hash), TimestampKeyPrefix); txErr != nil { // By timestamp
				return txErr
			}
			glog.Infof("Stored indexes (by timestamp) for %s - Height = %d", block.Hash, block.Height)

			if txErr = storeUIntIndex(txn, block.Height, []byte(block.Hash), HeightKeyPrefix); txErr != nil { // By block Height
				return txErr
			}
			glog.Infof("Stored indexes (by block´s height) for %s - Height = %d", block.Hash, block.Height)
		}

		return txErr
	})

	if err != nil {
		glog.Error(err)
	}

	return err
}

// GetBlock read a block from the database using their hash
func (s Store) GetBlock(hash string) (*types.FullSignedBlock, error) {

	var block types.FullSignedBlock
	err := s.storHandler.View(func(txn *badger.Txn) error {
		bytes, err := readStringIndex(txn, hash, HashKeyPrefix)
		if err != nil {
			return err
		}
		err = json.Unmarshal(bytes, &block)

		return err
	})

	return &block, err
}

// GetLatestBlocks returns n blocks starting in the specified timeouts. The algorithm will iterate over
// the database looking for stored timestamps lower or equal to the parameter.
func (s Store) GetLatestBlocks(timestamp uint64, n int) ([]types.FullSignedBlock, error) {

	var blocks []types.FullSignedBlock

	err := s.storHandler.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		var hashes []string
		for it.Rewind(); it.Valid(); it.Next() {
			// Check when the list of hashes is full
			if len(hashes) == n {
				break
			}
			item := it.Item()
			k := item.Key()

			if k[0] == TimestampKeyPrefix { // Check only the timestamp indexes
				storedTimestamp := binary.BigEndian.Uint64((k[1:]))
				if storedTimestamp <= timestamp {
					// Get the value
					value, err := item.ValueCopy(nil)
					if err != nil {
						return err
					}
					hashes = append(hashes, string(value))
				}
			}
		}
		// there are n hashes, get the blocks
		for i := 0; i < n; i++ {

			var block types.FullSignedBlock
			bytes, err := readStringIndex(txn, hashes[i], HashKeyPrefix)
			if err != nil {
				return err
			}
			err = json.Unmarshal(bytes, &block)
			blocks = append(blocks, block)
		}

		return nil
	})

	return blocks, err
}

// Read a block from the database using their timestamp as index
func (s Store) FindBlockByTimestamp(timestamp uint64) (*types.FullSignedBlock, error) {

	var block types.FullSignedBlock
	err := s.storHandler.View(func(txn *badger.Txn) error {
		// retrieve the the indexed hash
		hashBytes, err := readUIntIndex(txn, timestamp, TimestampKeyPrefix)
		if err != nil {
			return err
		}
		// Get the hash from the result and look for the block
		bytes, err := readStringIndex(txn, string(hashBytes), HashKeyPrefix)
		if err != nil {
			return err
		}
		err = json.Unmarshal(bytes, &block)

		return err
	})

	return &block, err
}

// Read a block from the database using their timestamp as index
func (s Store) FindBlockByHeight(height uint64) (*types.FullSignedBlock, error) {

	var block types.FullSignedBlock
	err := s.storHandler.View(func(txn *badger.Txn) error {
		// retrieve the the indexed hash
		hashBytes, err := readUIntIndex(txn, height, HeightKeyPrefix)
		if err != nil {
			return err
		}
		// Get the hash from the result and look for the block
		bytes, err := readStringIndex(txn, string(hashBytes), HashKeyPrefix)
		if err != nil {
			return err
		}
		err = json.Unmarshal(bytes, &block)

		return err
	})

	return &block, err
}

// StoreValue stores an abritrary value in the database, indexed by a string
func (s Store) StoreValue(key string, value []byte) error {

	err := s.storHandler.Update(func(txn *badger.Txn) error {
		return storeStringIndex(txn, key, value, FixedKeyPrefix)
	})

	return err
}

// GetValue returns a value stored in the database indexed by an string
func (s *Store) GetValue(key string) ([]byte, error) {

	var bytes []byte
	var err error

	err = s.storHandler.Update(func(txn *badger.Txn) error {
		bytes, err = readStringIndex(txn, key, FixedKeyPrefix)

		return err
	})

	return bytes, err
}
