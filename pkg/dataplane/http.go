package dataplane

import (
	"balanced/pkg/configuration"
	"balanced/pkg/types"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	defaultAPIVersion = "v2"

	configEndpoint         = "/services/haproxy/configuration/version"
	getBackendsEndpoint    = "/services/haproxy/configuration/backends"
	getServersEndpoint     = "/services/haproxy/configuration/servers"
	runtimeServersEndpoint = "/services/haproxy/runtime/servers"
)

type auth struct {
	username string
	password string
}

type DataPlaneAPIClient struct {
	auth    auth
	baseUrl *url.URL
	c       *http.Client
}

func (d *DataPlaneAPIClient) GetConfigVersion() (int, int, error) {
	reqUrl := d.baseUrl.JoinPath(configEndpoint)

	req, _ := d.newRequest(http.MethodGet, reqUrl.String(), nil)

	resp, err := d.c.Do(req)
	if err != nil {
		return -1, resp.StatusCode, err
	}

	// Always close the body to prevent resource leaks
	defer resp.Body.Close()

	return processResponse[int](resp)
}

func (d *DataPlaneAPIClient) GetOrCreateBackend(def *types.LoadBalancerUpstreamDefinition) (*Backend, error) {
	reqUrl := d.baseUrl.JoinPath(getBackendsEndpoint, def.Domain)

	req, _ := d.newRequest(http.MethodGet, reqUrl.String(), nil)

	log.Infof("retrieving backend: %s", def.Domain)

	resp, err := d.c.Do(req)
	if err != nil {
		return nil, err
	}

	// Always close the body to prevent resource leaks
	defer resp.Body.Close()

	var backend *Backend

	r, status, err := processResponse[VersionedResponsed[*Backend]](resp)

	backend = r.Data

	if status == http.StatusNotFound {
		log.Infof("backend for %s does not exist, createding", def.Domain)
		backend, _, err = d.createBackend(def)
	}

	if err != nil {
		return nil, err
	}

	return backend, nil
}

func (d *DataPlaneAPIClient) UpdateServers(def *types.LoadBalancerUpstreamDefinition) error {
	version, _, err := d.GetConfigVersion()
	if err != nil {
		return err
	}

	_, addedServers, removedServers := ServersFromUpstream(def)

	log.Infof("%s - adding: %d, removing: %d", def.Domain, len(addedServers), len(removedServers))

	for _, server := range addedServers {
		if err := d.addServerToBackend(def.Domain, server, version); err != nil {
			return err
		}

		version += 1
	}

	for _, serverName := range removedServers {
		if err := d.removeServerFromBackend(def.Domain, serverName, version); err != nil {
			return err
		}

		version += 1
	}

	return nil
}

func (d *DataPlaneAPIClient) addServerToBackend(backendName string, server *Server, version int) error {
	reqUrl := d.baseUrl.JoinPath(runtimeServersEndpoint)

	q := reqUrl.Query()
	q.Add("backend", backendName)
	q.Add("version", fmt.Sprintf("%d", version))
	reqUrl.RawQuery = q.Encode()

	jsonData, err := json.Marshal(server)
	if err != nil {
		return err
	}

	req, _ := d.newRequest(http.MethodPost, reqUrl.String(), bytes.NewReader(jsonData))

	resp, err := d.c.Do(req)
	if err != nil {
		return err
	}

	// Always close the body to prevent resource leaks
	defer resp.Body.Close()

	_, _, err = processResponse[*Server](resp)

	if err != nil {
		return err
	}

	return nil
}

func (d *DataPlaneAPIClient) removeServerFromBackend(backendName, serverName string, version int) error {
	reqUrl := d.baseUrl.JoinPath(runtimeServersEndpoint, serverName)

	q := reqUrl.Query()
	q.Add("backend", backendName)
	q.Add("version", fmt.Sprintf("%d", version))
	reqUrl.RawQuery = q.Encode()

	req, _ := d.newRequest(http.MethodDelete, reqUrl.String(), nil)

	resp, err := d.c.Do(req)
	if err != nil {
		return err
	}

	// Always close the body to prevent resource leaks
	defer resp.Body.Close()

	_, _, err = processResponse[string](resp)

	if err != nil {
		return err
	}

	return nil
}

func (d *DataPlaneAPIClient) createBackend(def *types.LoadBalancerUpstreamDefinition) (*Backend, int, error) {
	version, _, err := d.GetConfigVersion()
	if err != nil {
		return nil, -1, err
	}

	reqUrl := d.baseUrl.JoinPath(getBackendsEndpoint)

	q := reqUrl.Query()
	q.Add("version", fmt.Sprintf("%d", version))
	reqUrl.RawQuery = q.Encode()

	jsonData, err := json.Marshal(BackendFromUpstream(def))
	if err != nil {
		return nil, -1, err
	}

	req, _ := d.newRequest(http.MethodPost, reqUrl.String(), bytes.NewReader(jsonData))

	resp, err := d.c.Do(req)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	// Always close the body to prevent resource leaks
	defer resp.Body.Close()

	r, status, err := processResponse[*Backend](resp)

	if err != nil {
		return nil, status, err
	}

	return r, status, nil
}

func (d *DataPlaneAPIClient) GetServers(b *Backend) ([]*Server, error) {
	reqUrl := d.baseUrl.JoinPath(getServersEndpoint)

	q := reqUrl.Query()
	q.Add("backend", b.Name)
	reqUrl.RawQuery = q.Encode()

	req, _ := d.newRequest(http.MethodGet, reqUrl.String(), nil)

	resp, err := d.c.Do(req)
	if err != nil {
		return nil, err
	}

	// Always close the body to prevent resource leaks
	defer resp.Body.Close()

	r, _, err := processResponse[VersionedResponsed[[]*Server]](resp)
	if err != nil {
		return nil, err
	}

	return r.Data, nil
}

func (d *DataPlaneAPIClient) newRequest(method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(d.auth.username, d.auth.password)
	req.Header.Add("Content-Type", "application/json")

	return req, nil
}

func processResponse[T any](resp *http.Response) (T, int, error) {
	var v T

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respText, _ := io.ReadAll(resp.Body)
		return v, resp.StatusCode, fmt.Errorf("unexpected HTTP status code: %d -> %s", resp.StatusCode, respText)
	}

	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return v, resp.StatusCode, err
	}

	return v, resp.StatusCode, nil
}

func (d *DataPlaneAPIClient) validate(cfg configuration.HAProxyDataPlaneConfig) error {
	if cfg.Endpoint == "" {
		return errors.New("data plane endpoint not specified")
	}

	if cfg.Username == "" || cfg.Password == "" {
		return errors.New("data plane authentication credentials are not specified. Set both username and password fields")
	}

	d.auth = auth{
		username: cfg.Username,
		password: cfg.Password,
	}

	var version = cfg.Version
	if version == "" {
		version = defaultAPIVersion
	}

	var baseUrl string
	var err error

	if baseUrl, err = url.JoinPath(cfg.Endpoint, version); err != nil {
		return err
	}

	if d.baseUrl, err = url.Parse(baseUrl); err != nil {
		return errors.New("data plane endpoint is not a valid url: " + err.Error())
	}

	return nil
}

func NewDataPlaneAPIClient(cfg configuration.HAProxyDataPlaneConfig) (*DataPlaneAPIClient, error) {
	// 1. Configure the underlying Transport (Connection Pool & Dialing)
	transport := &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second, // Max time waiting for TCP connection
			KeepAlive: 30 * time.Second, // TCP keep-alive interval
		}).DialContext,

		// TLSClientConfig: &tls.Config{
		// 	MinVersion: tls.VersionTLS12,
		// },
	}

	// 2. Configure the HTTP Client Wrapper
	d := &DataPlaneAPIClient{
		c: &http.Client{
			Transport: transport,

			// Hard boundary for the entire request lifecycle (Dial + TLS + Request + Response Body)
			Timeout: 30 * time.Second,
		},
	}

	return d, d.validate(cfg)
}
