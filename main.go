package main

import (
	"log"
	"net/http"

	"github.com/crlsmrls/dummybox/cmd"
)

// get version from ENV variable VERSION
var Version = "development"

func main() {
	cmd.Version = Version

	dMux := http.NewServeMux()
	dMux.HandleFunc("/positions", cmd.PositionsHandler)
	dMux.HandleFunc("/version", cmd.VersionHandler)
	dMux.HandleFunc("/info", cmd.InfoHandler)

	go func() {
		log.Default().Println("Server running on port 8080")
		log.Fatal(http.ListenAndServe(":8080", dMux))
	}()

	select {}
}
