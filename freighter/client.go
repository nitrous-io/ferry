package freighter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
)

const USER_AGENT = "ferry"

type (
	Client interface {
		ContainerIP(identifier string) net.IP
		Refresh() error
	}

	DefaultClient struct {
		BaseURL      string
		ContainerMap map[string]string // Map of "container name/ID" -> "ip address"

		client *http.Client
	}

	ContainerResponse struct {
		ID      string `json:"id"`
		Name    string `json:"name`
		Host    string `json:"host`
		State   string `json:"state`
		Image   string `json:"image`
		SSHPort string `json:"ssh_port`
	}

	ContainersResponse struct {
		Containers []*ContainerResponse `json:"containers"`
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
		c.ContainerMap[container.ID] = container.Host
		c.ContainerMap[container.Name] = container.Host
	}
	log.Printf("Refreshed container info from Freighter: %v", c.ContainerMap)

	return err
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

	return body, nil
}

func (c *DefaultClient) getURL(path string) string {
	return fmt.Sprintf("%s/%s", c.BaseURL, path)
}

func NewClient(baseURL string) *DefaultClient {
	return &DefaultClient{
		BaseURL:      baseURL,
		ContainerMap: make(map[string]string),
		client:       &http.Client{},
	}
}
