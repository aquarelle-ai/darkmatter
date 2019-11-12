/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle AI
**/
package types

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"

	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"
)

const (
	ServiceHash = "1d0684170dcf58ed2499d233be72b5dde48d8124cb617f1309bae85da2fe85cf"

	ChainDataDir        = "./chain"
	LatestBlockFileName = ChainDataDir + "/latest-block.json"
)

// The model used to get the data
type PriceSummary struct {
	// Symbol string `json:"symbol"`
	// PriceChange        float32 `json:"priceChange"`
	// PriceChangePercent float32 `json:"priceChangePercent"`
	// LastQty            float32 `json:"LastQty"`
	// // LastPrice          float32 `json:"lastPrice"`
	// BidPrice           float32 `json:"bidPrice"`
	// AskPrice           float32 `json:"askPrice"`
	// BidQty             float32 `json:"bidQty"`
	// AskQty             float32 `json:"askQty"`
	QuoteVolume float64 `json:"quoteVolumen"`
	Volume      float64 `json:"volume"`
	HighPrice   float64 `json:"highPrice"`
	OpenPrice   float64 `json:"openPrice"`
	// LowPrice           float64 `json:"lowPrice"`
	// OpenTime           int64  `json:"openTime"`
	// CloseTime          int64  `json:"closeTime"`
}

func (info PriceSummary) String() string {
	result, err := json.Marshal(&info)

	if err != nil {
		panic(err)
	}

	return string(result)
}

// Message used to be send to users and index the blocks
type LiteIndexValueMessage struct {
	Height      int64   `json:"height"`
	PriceIndex  float64 `json:"priceIndex"`
	Quoted      string  `json:"quote"`
	NodeAddress string  `json:"nodeAddress"`
	Timestamp   int64   `json:"timestamp"`
}

/**************   Service´s Public Messages ******************/
// Message to send to the connected clients through websocket
type FullSignedBlock struct {
	Hash         string `json:"hash"`
	Height       int64  `json:"height"`
	Timestamp    int64  `json:"timestamp"`
	PreviousHash string `json:"previousHash"`
	MerkleRoot   string `json:"merkleRoot"`

	AveragePrice  float64  `json:"avgPrice"`
	AverageVolume float64  `json:"avgVolumen"`
	Ticker        string   `json:"ticker"`
	Sources       []Result `json:"sources"`
}

// Store the current last block in memory. Used to create the chain
var latestBlock *FullSignedBlock

// Return a new message with
func NewFullSignedBlock(uid string, ticker string, avgPrice float64, avgVolumen float64, sources []Result) FullSignedBlock {

	// Create a "protomessage" in order to be hashed with the hash inside
	var latestHash string
	var height int64

	if latestBlock == nil { // try to get the stored block
		latestBlock = readLatestBlock()
	}

	if latestBlock != nil {
		latestHash = latestBlock.Hash   // Yes, there is a latest block, so there is a "latest" of everything
		height = latestBlock.Height + 1 // And a new heigth
	}

	block := FullSignedBlock{
		Height:        height,
		AveragePrice:  avgVolumen,
		AverageVolume: avgPrice,
		Ticker:        ticker,
		Timestamp:     time.Now().Unix(),
		PreviousHash:  latestHash, // Chain the current hash with the previous one
		Sources:       sources,
	}

	bytes, err := json.Marshal(block)
	if err != nil {
		log.Println("Error serializing message", err)
	}
	// Sign the content of block including the hash of DarkMatter
	rawContent := fmt.Sprintf("%s:%s", ServiceHash, bytes)
	log.Println("Text to hash", rawContent)

	// Double hash for the content
	doubleHash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawContent)))
	doubleHash = fmt.Sprintf("%x", sha256.Sum256([]byte(doubleHash)))
	block.Hash = doubleHash

	latestBlock = &block
	// and keep it safe
	storeLatestBlock()

	return block
}

// Store the latest hash of the message
func storeLatestBlock() {

	// Check if exists the folder to store the block
	if _, err := os.Stat(ChainDataDir); os.IsNotExist(err) {
		err = os.Mkdir(ChainDataDir, os.ModeDir)
		if err != nil {
			log.Fatal(err)
		}
	}

	f, err := os.Create(LatestBlockFileName)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	bytes, err := json.Marshal(latestBlock)
	if err != nil {
		log.Fatal(err)
		return // Don´t continue
	}

	f.WriteString(string(bytes))
	f.Sync()
}

func readLatestBlock() *FullSignedBlock {

	// Check if exists the folder to store the block
	if _, err := os.Stat(LatestBlockFileName); os.IsNotExist(err) {
		log.Fatal(err)
		return nil
	}

	f, err := os.Open(LatestBlockFileName)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	defer f.Close()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	var block FullSignedBlock
	err = json.Unmarshal(bytes, &block)
	if err != nil {
		log.Fatal(err)
		return nil // Don´t continue
	}

	return &block
}

/**************** Map & Reduce types ***************************/
// The job message to insert in a queue to be processed as part of the the Mapping Stage
type GetDataJob struct {
	Quote       string
	UID         string
	DataCrawler PriceSourceCrawler
}

// The result message that will receive the results from the mapped nodes in the Reduce Stage
type Result struct {
	CrawlerName string       `json:"name"`
	Data        PriceSummary `json:"data"`
	UID         string       `json:"messageUid"`
	HasError    bool         `json:"hasError"`
	Timestamp   int64        `json:"timestamp"`
	Ticker      string       `json:"ticker"`
}

/*****  Crawkers types **********************/
// Interface for clients
type PriceSourceCrawler interface {
	Crawl(quotedCurrency string, done chan PriceSummary)
	GetName() string
	GetTicker() string
}

// GenerateRandomBytes returns securely generated random bytes. It will return an error if the system's secure random
// number generator fails to function correctly, in which case the caller should not continue.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return b, nil
}

// GenerateRandomString returns a URL-safe, base64 encoded securely generated random string.
// It will return an error if the system's secure random  number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomString(s int) (string, error) {
	b, err := GenerateRandomBytes(s)
	if err != nil {
		log.Fatal(err)
	}
	return base64.URLEncoding.EncodeToString(b), err
}
