package main

import (
	"log"

	"github.com/mannemsolutions/pgfga/internal"
)

func main() {
	fga, err := internal.NewPgFgaHandler()
	if err != nil {
		log.Fatalf("Error occurred on getting config: %e", err)
	}

	fga.Handle()
}
