// Package database include the main functions and type to manage the blockchain, the blocks and their storage
package database

import (
	"encoding/json"
	"log"
	"time"

	"github.com/aquarelle-tech/darkmatter/types"
)

const (
	// PublicRootDir is the directory to publish the static files ofr HTTP requests
	PublicRootDir = "public"
	// LatestBlockKey is the literal to be used as a key to index the latest block in the database
	LatestBlockKey = "latest"
	// BlockchainFileLocation is the directory where to store the database for the node
	BlockchainFileLocation = "./chain/stor"

	// MainBlockChainName is the default name for the main blocks chain
	MainBlockChainName = "main"
)

var (
	// PublicBlockDatabase is the main instance to manage the database
	PublicBlockDatabase *BlockChain = NewBlockChain(MainBlockChainName, BlockchainFileLocation)
)

// BlockChain is the main data model to handle the blocks
type BlockChain struct {
	Name      string
	IsTestnet bool

	latestBlock *types.FullSignedBlock
	Store       types.KVStore
}

// NewBlockChain initializes and creates a new manager of a blockchain
func NewBlockChain(name string, locationDirectory string) *BlockChain {
	return &BlockChain{
		Name:  name,
		Store: NewKVStore(locationDirectory),
	}
}

// NewFullSignedBlock creates a new signed block to store
func (db *BlockChain) NewFullSignedBlock(ticker string, avgPrice float64, avgVolumen float64, sources []types.Result, memo string) types.FullSignedBlock {

	// Create a "protomessage" in order to be hashed with the hash inside
	var latestHash string
	var height uint64

	if db.latestBlock == nil { // try to get the stored block
		db.ReadLatestBlock()
	}

	if db.latestBlock != nil {
		latestHash = db.latestBlock.Hash   // Yes, there is a latest block, so there is a "latest" of everything
		height = db.latestBlock.Height + 1 // And a new heigth
	}

	block := types.FullSignedBlock{
		Height:        height,
		AveragePrice:  avgVolumen,
		AverageVolume: avgPrice,
		Ticker:        ticker,
		Timestamp:     uint64(time.Now().Unix()),
		PreviousHash:  latestHash, // Chain the current hash with the previous one
		Evidence:      sources,
		Memo:          memo,
	}
	// Other settings
	block.CreateHash()
	if db.latestBlock != nil {
		block.PreviousAddress = db.latestBlock.Address // Link with previous block
	}

	db.Store.StoreBlock(block)
	// Latest block
	db.latestBlock = &block
	bytes, err := json.Marshal(block)
	if err != nil {
		panic(err) //TODO: This error is important!! means that there was not able to create a new block! Needs more code to manage this event
	}
	db.Store.StoreValue(LatestBlockKey, bytes)

	// log.Println("Created a new block", block)
	return block
}

// Store the latest hash of the message
func (db *BlockChain) StoreLatestBlock() {

	bytes, err := json.Marshal(db.latestBlock)
	if err != nil {
		log.Println("Can´t store the latest block. Please check the Store urgently!!")
		return // Don´t continue
	}
	db.Store.StoreValue(LatestBlockKey, bytes)
}

// Get the latest stored block
func (db *BlockChain) ReadLatestBlock() {

	bytes, err := db.Store.GetValue(LatestBlockKey)
	if err != nil {
		log.Println("The repository for the latest block don´t exists. Is is the genesis block?")
		return
	}

	var block types.FullSignedBlock
	err = json.Unmarshal(bytes, &block)
	if err != nil {
		log.Println("The latest block is corrupt or invalid")
		return
	}

	db.latestBlock = &block
}
