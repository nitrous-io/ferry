package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/nitrous-io/ferry/freighter"
	"github.com/nitrous-io/ferry/server"
)

var version string

var (
	port        = flag.Uint("p", 65343, "Specify the port on which ferry listens for connections")
	showVersion = flag.Bool("v", false, "Display the version and exit")

	envFreighterBaseURL = os.Getenv("FREIGHTER_BASE_URL")
	freighterBaseURL    = "http://127.0.0.1:65333"

	envDockerHost = os.Getenv("DOCKER_HOST")
	dockerHost    = "unix:///var/run/docker.sock"
)

func init() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("ferry: version %v\n", version)
		os.Exit(0)
	}

	if envFreighterBaseURL != "" {
		freighterBaseURL = envFreighterBaseURL
	}

	if envDockerHost != "" {
		dockerHost = envDockerHost
	}
}

func main() {
	freighterClient := freighter.NewClient(freighterBaseURL, dockerHost)
	server := server.NewServer(freighterClient, 5*time.Second) // TODO Make timeout configurable.
	host := "0.0.0.0"
	server.Start(host, uint16(*port))
}
