package main

import (
	"context"
	"flag"
	"log"
	"pictionary/pkg/server"
)

func main() {
	path := flag.String("path", "", "Path to images directory.")
	flag.Parse()
	if *path == "" {
		log.Fatal("Please, provide -path to images dir.")
	}
	server := &server.Server{":5555", *path, 1000}

	server.Start(context.Background())
}
