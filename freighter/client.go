package freighter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/nitrous-io/ferry/util"
)

const userAgent = "ferry"

type ContainerNotFoundError string

func (e ContainerNotFoundError) Error() string {
	return "container not found " + string(e)
}

type (
	Client interface {
		ContainerIP(identifier string) net.IP
		Update(id string) error
		Refresh() error
		StartEventListener()
		StopEventListener()
	}

	DefaultClient struct {
		Socket       string
		ContainerMap map[string]string // Map of "container name/ID" -> "ip address"

		nameMap           map[string]string // Internally used map of "container ID" -> "container name"
		dockerHost        string
		quitEventListener chan bool
		client            *http.Client
	}

	Container struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Host    string `json:"host"`
		State   string `json:"state"`
		Image   string `json:"image"`
		SSHPort int    `json:"ssh_port"`
	}

	ContainerResponse struct {
		*Container `json:"container"`
	}

	ContainersResponse struct {
		Containers []*Container `json:"containers"`
	}
)

// ContainerIP returns the IP address of a container given its identifier,
// which can be either its ID or name.
// If container is not found, returns nil.
func (c *DefaultClient) ContainerIP(identifier string) net.IP {
	if ip, ok := c.ContainerMap[identifier]; ok {
		return net.ParseIP(ip)
	}
	return nil
}

// Update queries a Freighter server for the container with given id and updates
// its ContainerMap with the container's IP address.
func (c *DefaultClient) Update(id string) error {
	data, err := c.get(fmt.Sprintf("/containers/%s", id))
	if err != nil {
		if _, ok := err.(ContainerNotFoundError); ok {
			c.RemoveContainer(id)
			return nil
		}
		return err
	}

	var cResp ContainerResponse
	if err := json.Unmarshal(data, &cResp); err != nil {
		return err
	}
	c.AddOrUpdateContainer(cResp.Container)

	return nil
}

// Refresh queries a Freighter server for the latest container information and
// updates its ContainerMap with the mapping of:
//   "container name/ID" -> "ip address".
func (c *DefaultClient) Refresh() error {
	// TODO There only needs to be 1 refresh request running at any one time.
	data, err := c.get("/containers")
	if err != nil {
		return err
	}
	var containers ContainersResponse
	if err := json.Unmarshal(data, &containers); err != nil {
		return err
	}

	// Re-initialize ContainerMap from response.
	c.ContainerMap = make(map[string]string)
	for _, container := range containers.Containers {
		c.AddOrUpdateContainer(container)
	}

	return err
}

// StartEventListener starts monitoring events like container start and stop.
// The ContainerMap is updated on these events so that our IP address mapping
// stays up-to-date.
// TODO We probably should not assume that we are using Docker.
func (c *DefaultClient) StartEventListener() {
	docker, err := dockerclient.NewClient(c.dockerHost)
	if err != nil {
		util.Logger.Warnf("Unable to connect to Docker daemon: %v", err)
		go c.startActivePolling(2 * time.Second)
		return
	}
	listener := make(chan *dockerclient.APIEvents)
	if err = docker.AddEventListener(listener); err != nil {
		util.Logger.Warnf("Cannot add Docker event listener: %v", err)
		go c.startActivePolling(2 * time.Second)
		return
	}

	go func() {
		for {
			select {
			case event := <-listener:
				if event.Status == "start" || event.Status == "die" {
					util.Logger.Debugf("Got Docker event: %v", event)
					c.Update(event.ID)
				}
			case <-c.quitEventListener:
				util.Logger.Info("Quitting Docker event monitor")
				docker.RemoveEventListener(listener)
				break
			}
		}
	}()
}

// StopEventListener stops the monitoring of container events.
func (c *DefaultClient) StopEventListener() {
	c.quitEventListener <- true
}

func (c *DefaultClient) AddOrUpdateContainer(container *Container) {
	if container.State == "running" && container.Host != "" {
		c.ContainerMap[container.ID] = container.Host
		c.nameMap[container.ID] = container.Name
		c.ContainerMap[container.Name] = container.Host
	} else {
		c.RemoveContainer(container.ID)
	}
}

func (c *DefaultClient) RemoveContainer(id string) {
	delete(c.ContainerMap, id)
	if name, ok := c.nameMap[id]; ok {
		delete(c.ContainerMap, name)
	}
}

func (c *DefaultClient) startActivePolling(interval time.Duration) {
	util.Logger.Debug("Starting active polling for container changes")
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			if err := c.Refresh(); err != nil {
				util.Logger.Warnf("Failed to get container changes: %v", err)
			}
		case <-c.quitEventListener:
			util.Logger.Info("Quitting active polling")
			break
		}
	}
}

func (c *DefaultClient) get(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	req.URL.Host = c.Socket
	req.URL.Scheme = "http"
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("freighter: request to API failed: %v", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return body, ContainerNotFoundError(fmt.Sprintf("GET %s", req.URL))
	}

	return body, nil
}

// NewClient returns a client that:
// - communicates with freighterd over an unix socket
// - communicates with dockerHost with the help of go-dockerclient
func NewClient(freighterEndpoint, dockerHost string) *DefaultClient {
	tp := &http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.Dial("unix", freighterEndpoint)
		},
	}
	return NewClientTransport(freighterEndpoint, dockerHost, tp)
}

// NewClienTransport is internally used in this pkg, and replace the default
// http client transport with the desired transport if needed
func NewClientTransport(freighterEndpoint, dockerHost string, tp *http.Transport) *DefaultClient {
	client := &http.Client{}
	if tp != nil {
		client.Transport = tp
	}
	return &DefaultClient{
		Socket:            freighterEndpoint,
		dockerHost:        dockerHost,
		ContainerMap:      make(map[string]string),
		nameMap:           make(map[string]string),
		client:            client,
		quitEventListener: make(chan bool),
	}
}
