package main

import (
	"github.com/xackery/grapheq/webserver"
	"log"
	"os"
)

func main() {
	err := webserver.Start("localhost:12345")
	if err != nil {
		log.Println("Error with webserver:", err.Error())
		os.Exit(1)
	}
}
