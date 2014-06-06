package freighter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	dockerclient "github.com/fsouza/go-dockerclient"
)

const USER_AGENT = "ferry"

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
		BaseURL      string
		ContainerMap map[string]string // Map of "container name/ID" -> "ip address"

		nameMap           map[string]string // Internally used map of "container ID" -> "container name"
		dockerHost        string
		quitEventListener chan bool
		client            *http.Client
	}

	Container struct {
		ID      string `json:"id"`
		Name    string `json:"name`
		Host    string `json:"host`
		State   string `json:"state`
		Image   string `json:"image`
		SSHPort string `json:"ssh_port`
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
	data, err := c.get(fmt.Sprintf("containers/%s", id))
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
	data, err := c.get("containers")
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
		log.Printf("Unable to connect to Docker daemon: %v", err)
		go c.startActivePolling(2 * time.Second)
		return
	}
	listener := make(chan *dockerclient.APIEvents)
	// NOTE: Docker 0.11.1 has a bug where only a single Docker event listener
	// gets changes. Fixed in https://github.com/dotcloud/docker/pull/5827.
	if err = docker.AddEventListener(listener); err != nil {
		go c.startActivePolling(2 * time.Second)
		return
	}

	go func() {
		for {
			select {
			case event := <-listener:
				if event.Status == "start" || event.Status == "stop" {
					c.Update(event.ID)
				}
			case <-c.quitEventListener:
				log.Print("Quitting Docker event monitor")
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
	log.Printf("Starting active polling for container changes")
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			if err := c.Refresh(); err != nil {
				log.Printf("Failed to get container changes: %v", err)
			}
		case <-c.quitEventListener:
			log.Print("Quitting active polling")
			break
		}
	}
}

func (c *DefaultClient) get(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", c.getURL(path), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", USER_AGENT)
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

func (c *DefaultClient) getURL(path string) string {
	return fmt.Sprintf("%s/%s", c.BaseURL, path)
}

func NewClient(baseURL, dockerHost string) *DefaultClient {
	return &DefaultClient{
		BaseURL:           baseURL,
		dockerHost:        dockerHost,
		ContainerMap:      make(map[string]string),
		nameMap:           make(map[string]string),
		client:            &http.Client{},
		quitEventListener: make(chan bool),
	}
}
