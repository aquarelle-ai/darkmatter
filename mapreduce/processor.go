/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle AI
**/
package mapreduce

import (
	"sync"
	"time"

	"aquarelle-tech/darkmatter/database"
	"aquarelle-tech/darkmatter/types"
)

const (
	// How many seconds between a call and another one
	DELAY_BETWEEN_CRAWLS = 2 * time.Second
)

var publicBlockDatabase database.BlockChain = database.BlockChain{}

type Processor struct {
	// Channels to build the worker pool
	DataJobs chan types.GetDataJob
	Results  chan types.Result

	Directory       []types.PriceEvidenceCrawler
	QuotedCurrency  string
	PublicationChan chan types.FullSignedBlock
}

func NewMapReduceProcessor(directory []types.PriceEvidenceCrawler, quotedCurrency string, publicationChan chan types.FullSignedBlock) Processor {
	// Channels to build the worker pool
	return Processor{
		Directory:       directory,
		QuotedCurrency:  quotedCurrency,
		PublicationChan: publicationChan,
	}
}

// Collect the results
func (p Processor) mapJob(wg *sync.WaitGroup) {

	// The process should also be called async
	internalChan := make(chan types.QuotePriceInfo)

	for job := range p.DataJobs {
		// Get the data
		go job.DataCrawler.Crawl(job.Quote, internalChan)

		data := <-internalChan
		result := types.Result{
			Data:        data,
			Ticker:      job.DataCrawler.GetTicker(),
			HasError:    false,
			Timestamp:   time.Now().Unix(),
			CrawlerName: job.DataCrawler.GetName(),
		}
		result.CreateHash()

		// Send the result to the queue
		p.Results <- result
	}

	close(internalChan)
	wg.Done()
}

func (p Processor) createWorkerPool(size int) {
	var wg sync.WaitGroup

	for i := 0; i < size; i++ {
		wg.Add(1)
		go p.mapJob(&wg)
	}
	wg.Wait()

	close(p.Results)
}

// Creates the full list of jobs for each crawler in the directory
func (p Processor) allocateJobs(poolSize int) {
	for i := 0; i < poolSize; i++ {
		newJob := types.GetDataJob{
			Quote:       p.QuotedCurrency,
			DataCrawler: p.Directory[i], // Get the crawler
		}
		p.DataJobs <- newJob
	}

	close(p.DataJobs)
}

// Execute the Reduce stage. Get all the data crawled from the sources and generates an aggregate index
func (p Processor) reduceJobs(poolSize int) {
	var totalVolume float64
	// var totalQuoted float64
	var totalPrice float64

	//====================  HACK: This code must be replaced with the real algorithm to calculate the avg price ======
	ticker := "BTCUSD"

	var sources []types.Result
	// NOTE: Instead of sum or any other calculation, the code will below will use a value from any of the providers, temporarly
	for result := range p.Results {
		sources = append(sources, result)
	}

	// Get the first
	totalVolume = sources[0].Data.Volume
	totalPrice = sources[0].Data.HighPrice
	//====================================================================================================================

	// Create a message to send to service´s listeners
	newMsg := publicBlockDatabase.NewFullSignedBlock(
		ticker,
		totalPrice,  // Average price
		totalVolume, // High price
		sources,
	)

	p.PublicationChan <- newMsg
}

func (p Processor) mapReduceLoop() {
	poolSize := len(p.Directory)
	for {
		// Channels to build the worker pool
		p.DataJobs = make(chan types.GetDataJob, poolSize)
		p.Results = make(chan types.Result, poolSize)

		// Create the jobs an launch the process to create
		go p.allocateJobs(poolSize)
		go p.reduceJobs(poolSize)

		p.createWorkerPool(poolSize)
		// and wait to request a new block of daya
		time.Sleep(DELAY_BETWEEN_CRAWLS)
	}
}

// Launch the main loop of the map-reduce processor. The method verify the data before to launch the main loop
func (p Processor) Initialize() {
	//TODO: Validate the parameterized data
	go p.mapReduceLoop()
}
