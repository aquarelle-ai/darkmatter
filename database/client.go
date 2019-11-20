package database

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"aquarelle-tech/darkmatter/types"
)

const (
	PublicRootDir = "public"
	// Starting directory where the data is written
	RootDataDir = "chain"

	// Starting directory where all the blocks are stored
	BlocksDataDir = RootDataDir + "/blocks"

	// Full filename (including path) for the file where the latest block is stored
	LatestBlockFileName = RootDataDir + "/latest-block.json"
)

type BlockChain struct {
	latestBlock *types.FullSignedBlock
}

// Creates a new signed block to store
func (db *BlockChain) NewFullSignedBlock(ticker string, avgPrice float64, avgVolumen float64, sources []types.Result) types.FullSignedBlock {

	// Create a "protomessage" in order to be hashed with the hash inside
	var latestHash string
	var height int64

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
		Timestamp:     time.Now().Unix(),
		PreviousHash:  latestHash, // Chain the current hash with the previous one
		Evidence:      sources,
	}
	// Other settings
	block.CreateHash()
	if db.latestBlock != nil {
		block.PreviousAddress = db.latestBlock.Address // Link with previous block
	}

	db.StoreBlock(&block)
	// Latest block
	db.latestBlock = &block
	db.StoreLatestBlock()

	log.Println("Created a new block", block)
	return block
}

// Store a block inside the public repository
func (db *BlockChain) StoreBlock(block *types.FullSignedBlock) {

	CheckRootDir()

	t := time.Unix(block.Timestamp, 0)
	// The full path where to store the block
	repositoryPath := fmt.Sprintf("%s/%d/%d/%d/%d/%d", BlocksDataDir, t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute())
	// Use the hash to name the filename
	blockFileName := fmt.Sprintf("%s/%s.json", repositoryPath, block.Hash)
	// Create the folder structure
	fullRepositoryPath := fmt.Sprintf("./%s/%s", PublicRootDir, repositoryPath)
	if _, err := os.Stat(fullRepositoryPath); os.IsNotExist(err) {
		err = os.MkdirAll(fullRepositoryPath, 0755)
		if err != nil {
			log.Fatal("Error creating blocks repository.", err)
			panic(err) // if the directory can´t be created, the application must stop!
		}
	}

	fullBlockFileName := fmt.Sprintf("./%s/%s", PublicRootDir, blockFileName)
	f, err := os.Create(fullBlockFileName)
	if err != nil {
		log.Fatal("Can´t create the block file.", err)
		panic(err) // if the block can´t be stored, the application should stop!
	}

	defer f.Close()

	// Set the address of the block. IMPORTANT, This value is not included in the hash
	//TODO: Get the server address from a config file
	block.Address = blockFileName
	// Serialize all the block
	bytes, err := json.MarshalIndent(block, "", " ")
	f.WriteString(string(bytes))
	f.Sync()
	log.Println("Stored block ", block.Hash)
}

// Store the latest hash of the message
func (db *BlockChain) StoreLatestBlock() {

	CheckRootDir()

	latestBlockFullFileName := fmt.Sprintf("./%s/%s", PublicRootDir, LatestBlockFileName)
	f, err := os.Create(latestBlockFullFileName)
	if err != nil {
		log.Fatal("Can´t create the config repository file", err)
	}
	log.Println("Created/Updated repository file", LatestBlockFileName)

	defer f.Close()

	bytes, err := json.Marshal(db.latestBlock)
	if err != nil {
		log.Fatal(err)
		return // Don´t continue
	}

	f.WriteString(string(bytes))
	f.Sync()
}

// Get the latest stored block
func (db *BlockChain) ReadLatestBlock() {

	// Check if exists the folder to store the block. if not exists, the method will return silently
	latestBlockFullFileName := fmt.Sprintf("./%s/%s", PublicRootDir, LatestBlockFileName)
	if _, err := os.Stat(latestBlockFullFileName); os.IsNotExist(err) {
		log.Println("The repository for the latest block don´t exists. Is is the genesis block?")
		return
	}

	f, err := os.Open(latestBlockFullFileName)
	if err != nil {
		log.Fatal(err)
		return
	}

	defer f.Close()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Println("Loaded the latest block from the repository.")

	var block types.FullSignedBlock
	err = json.Unmarshal(bytes, &block)
	if err != nil {
		log.Println(err)
		return // Don´t continue
	}
	db.latestBlock = &block
}

// Check if exists the folder to store the block
func CheckRootDir() error {
	dataRootDir := fmt.Sprintf("./%s/%s", PublicRootDir, RootDataDir)
	if _, err := os.Stat(dataRootDir); os.IsNotExist(err) {
		log.Println("Creating repository directory.", RootDataDir)

		err = os.MkdirAll(dataRootDir, os.ModeDir)
		if err != nil {
			log.Fatal("Error creating config repository.", err)
			return err
		}
	}

	return nil
}
