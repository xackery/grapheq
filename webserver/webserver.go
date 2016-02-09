package webserver

import (
	"log"
	"net/http"
)

func Start(addr string) (err error) {
	http.Handle("/", http.FileServer(assetFS()))
	log.Println("Started Web Server on", addr)
	err = http.ListenAndServe(addr, nil)
	return
}
