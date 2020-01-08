/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle Tech
**/

// Package cryptocerts
package cryptocerts

import (
	"sync"
	"time"
	"unsafe"

	"github.com/aquarelle-tech/darkmatter/service"
	cctypes "github.com/aquarelle-tech/darkmatter/x/cryptocerts/internal/types"

	"github.com/golang/glog"
)

var (
	// DelayBetweenBlocks set how many seconds should wait between a call and another one
	// TODO: Add this as a parameter in a external file
	DelayBetweenBlocks = time.Millisecond*1000 + time.Millisecond*500
)

// JobsProcessor will manage the jobs to crawl the data and the coordination between the process
type JobsProcessor struct {
	// Channels to build the worker pool
	DataJobs                  chan cctypes.GetDataJob
	Results                   chan cctypes.Result
	EvidenceCrawlersDirectory []cctypes.PriceEvidenceCrawler
	Publication               chan cctypes.QuotePriceData
}

// NewJobsProcessor is a constructor for a Processor
func NewJobsProcessor(directory []cctypes.PriceEvidenceCrawler, publication chan cctypes.QuotePriceData) JobsProcessor {
	// Channels to build the worker pool
	return JobsProcessor{
		EvidenceCrawlersDirectory: directory,
		Publication:               publication,
	}
}

// Collect the results
func (p JobsProcessor) mapJob(wg *sync.WaitGroup) {

	glog.Info("Processor: starting the loop for map all jobs")
	for job := range p.DataJobs {
		// Get the data
		// go job.DataCrawler.Crawl(internal)

		crawlerID := job.DataCrawler.GetUID()
		crawledData := <-p.Publication // Get the data
		glog.Infof("Read data from crawler '%s': %d bytes", crawlerID, unsafe.Sizeof(crawledData))

		result := cctypes.Result{
			Data:        crawledData,
			Timestamp:   time.Now().Unix(),
			CrawlerName: crawlerID,
		}

		result.UpdateHash()

		// Send the result to the queue
		p.Results <- result
	}

	wg.Done()
}

func (p JobsProcessor) createWorkerPool(size int) {
	var wg sync.WaitGroup

	glog.Infof("Processor: Creating workers pool: size=%d", size)
	for i := 0; i < size; i++ {
		wg.Add(1)
		go p.mapJob(&wg)
	}
	wg.Wait()

	close(p.Results)
}

// Creates the full list of jobs for each crawler in the directory
func (p JobsProcessor) allocateJobs(poolSize int) {
	for i := 0; i < poolSize; i++ {
		newJob := cctypes.GetDataJob{
			DataCrawler: p.EvidenceCrawlersDirectory[i], // Get the crawler
		}
		p.DataJobs <- newJob
	}

	close(p.DataJobs)
}

// Execute the Reduce stage. Get all the data crawled from the sources and generates an aggregate index
func (p JobsProcessor) reduceJobs() {
	// //====================  HACK: This code must be replaced with the real algorithm to calculate the avg price ======

	glog.Info("Processor: Getting data from jobs (reduce)")

	var volume int64
	var price float64
	var count int
	var evidenceList []cctypes.QuotePriceEvidence

	// NOTE: Instead of sum or any other calculation, the code will below will use a value from any of the providers, temporarly

	for result := range p.Results {

		quoteData := result.Data
		if (quoteData.BidQty + quoteData.AskQty) > 0 { // if there is a price level
			volume += (quoteData.BidVolume + quoteData.AskVolume)
			price += (quoteData.Bid + quoteData.Ask)

			// glog.Infof("Processor: Price component: price=%f, qty=%f", sumPrice, qty)

			evidenceList = append(evidenceList, quoteData.Evidence...)
			count++
		}
	}

	priceAvg := price / float64(volume)
	newPriceMsg := cctypes.QuotePriceMessage{
		AveragePrice: priceAvg,
		Volume:       int64(volume),
		Timestamp:    time.Now().Unix(),
	}

	glog.Infof("Processor: New average price from the aggregation of %d messages: %f", count, priceAvg)
	// Create a message to send to service´s listeners
	service.ServerInstance.AddBlock(newPriceMsg, evidenceList)
}

func (p JobsProcessor) mapReduceLoop() {
	poolSize := len(p.EvidenceCrawlersDirectory)

	for {
		// Channels to build the worker pool
		p.DataJobs = make(chan cctypes.GetDataJob, poolSize)
		p.Results = make(chan cctypes.Result, poolSize)

		// Create the jobs an launch the process to create
		go p.allocateJobs(poolSize)
		go p.reduceJobs()

		p.createWorkerPool(poolSize)
		// and wait to request a new block of daya
		time.Sleep(DelayBetweenBlocks)
		glog.Flush() // Clean the log cache

	}
}

// Initialize will launch the main loop of the map-reduce processor. The method verify the data before to launch the main loop
func (p JobsProcessor) Initialize() {
	//TODO: Validate the parameterized data
	go p.mapReduceLoop()
}
