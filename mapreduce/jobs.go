/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle AI
**/
package mapreduce

import (
	"sync"
	"time"

	"cratos.network/darkmatter/types"
	"github.com/google/uuid"
)

const (
	// How many seconds between a call and another one
	DELAY_BETWEEN_CRAWLS = 2 * time.Second
)

// Channels to build the worker pool
var getDataJobs chan types.GetDataJob
var results chan types.Result

// Collect the results
func mapJob(wg *sync.WaitGroup) {

	// The process should also be called async
	internalChan := make(chan types.PriceSummary)

	for job := range getDataJobs {
		// Get the data
		go job.DataCrawler.Crawl(job.Quote, internalChan)

		data := <-internalChan
		results <- types.Result{
			Data:        data,
			UID:         job.UID,
			Ticker:      job.DataCrawler.GetTicker(),
			HasError:    false,
			Timestamp:   time.Now(),
			CrawlerName: job.DataCrawler.GetName(),
		}
	}

	close(internalChan)
	wg.Done()
}

func createWorkerPool(size int) {
	var wg sync.WaitGroup
	for i := 0; i < size; i++ {
		wg.Add(1)
		go mapJob(&wg)
	}
	wg.Wait()
	close(results)
}

func allocate(poolSize int, quotedCurrency string, directory []types.PriceSourceCrawler) {
	newUID := uuid.New().String()
	for i := 0; i < poolSize; i++ {
		newJob := types.GetDataJob{
			Quote:       quotedCurrency,
			UID:         newUID,
			DataCrawler: directory[i], // Get the crawler
		}
		getDataJobs <- newJob
	}

	close(getDataJobs)
}

func reduceJobs(poolSize int, publicationChan chan types.PriceMessage) {
	var totalVolume float64
	var totalQuoted float64
	var totalPrice float64

	// fmt.Println("=====================================================")
	var uid string
	var ticker string
	for result := range results {
		totalVolume += result.Data.Volume
		totalQuoted += result.Data.QuoteVolume
		totalPrice += result.Data.HighPrice
		uid = result.UID
		ticker = "BTCUTC"
	}

	// Create a message to send to service´s listeners
	msg := types.PriceMessage{
		AveragePrice:  totalPrice / float64(poolSize),
		AverageVolume: (totalVolume / float64(poolSize)) / (totalQuoted / float64(poolSize)),
		UID:           uid,
		Ticker:        ticker,
		Timestamp:     time.Now().Unix(),
	}
	publicationChan <- msg
}

func MapReduceLoop(directory []types.PriceSourceCrawler, quotedCurrency string, publicationChan chan types.PriceMessage) {
	poolSize := len(directory)
	for {
		// Channels to build the worker pool
		getDataJobs = make(chan types.GetDataJob, poolSize)
		results = make(chan types.Result, poolSize)

		// Create the jobs an launch the process to create
		go allocate(poolSize, quotedCurrency, directory)
		go reduceJobs(poolSize, publicationChan)

		createWorkerPool(poolSize)
		time.Sleep(DELAY_BETWEEN_CRAWLS)
	}
}
