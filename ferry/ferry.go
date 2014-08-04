package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nitrous-io/ferry/freighter"
	"github.com/nitrous-io/ferry/server"
)

var version string

var (
	port        = flag.Uint("p", 65343, "Specify the port on which ferry listens for connections")
	showVersion = flag.Bool("v", false, "Display the version and exit")

	envFreighterSocket = os.Getenv("FREIGHTER_SOCKET")
	freighterSocket    = "/var/run/freighterd.sock"

	envDockerHost = os.Getenv("DOCKER_HOST")
	dockerHost    = "unix:///var/run/docker.sock"
)

func init() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("ferry: version %v\n", version)
		os.Exit(0)
	}

	if envFreighterSocket != "" {
		freighterSocket = envFreighterSocket
	}

	if envDockerHost != "" {
		dockerHost = envDockerHost
	}
}

func main() {
	log.Printf("setting up listener for freighter %s and docker %s",
		freighterSocket, dockerHost)
	freighterClient := freighter.NewClient(freighterSocket, dockerHost)
	server := server.NewServer(freighterClient, 5*time.Second) // TODO Make timeout configurable.
	host := "0.0.0.0"
	server.Start(host, uint16(*port))
}
