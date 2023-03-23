package main

import (
	"context"
	"log"

	"github.com/carlosmiguelsoto/oiajudge/pkg/cmsbridge"
	"github.com/carlosmiguelsoto/oiajudge/pkg/oiajudge"
)

func main() {
	bridge, err := cmsbridge.CreateCmsBridge()
	if err != nil {
		log.Fatal(err)
	}
	err = oiajudge.RunServer(context.Background(), bridge)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Bye.")
}
