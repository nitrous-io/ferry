package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/nitrous-io/ferry/server"
)

var version string

var (
	port        = flag.Uint("p", 65343, "Specify the port on which ferry listens for connections")
	showVersion = flag.Bool("v", false, "Display the version and exit")
)

func init() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("ferry: version %v\n", version)
		os.Exit(0)
	}
}

func main() {
	server := server.NewServer()
	http.Handle("/", server)

	host := "0.0.0.0"
	log.Printf("ferry: listening on %s:%d", host, *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", host, *port), nil))
}
