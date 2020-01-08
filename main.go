/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle Tech
**/
package main

import (
	// "encoding/binary"
	// "fmt"

	"flag"
	"net/http"

	"github.com/aquarelle-tech/darkmatter/x/cryptocerts"

	"github.com/aquarelle-tech/darkmatter/service"
	"github.com/golang/glog"
	// "github.com/aquarelle-tech/darkmatter/shamir"
)

func main() {

	flag.Parse()
	// s := uint64(123456)

	// buf := make([]byte, 8)
	// binary.BigEndian.PutUint64(buf, s)
	// fmt.Printf("buf=%d\n", buf)

	// result, _ := shamir.Split(buf, 5, 3)
	// fmt.Printf("result=%d\n", result)

	// bytes, _ := shamir.Combine(result)
	// fmt.Printf("bytes=%d\n", bytes)

	// fmt.Println(binary.BigEndian.Uint64(bytes))

	// quotedCurrency := "USD"

	// Prepare and run the subroutines for the oracle service
	server := service.NewOracleServer()
	server.Initialize()

	// Start the module
	cryptocerts.InitModule()

	// // Prepare and start the subroutines to manage the request of sources
	// processor := mapreduce.NewJobsProcessor(certs.Directory, certs.PublishedPrice)
	// processor.Initialize()

	err := http.ListenAndServe(":6877", nil)
	if err != nil {
		glog.Fatalf("Error initializing the server: %v", err)
	}

}
