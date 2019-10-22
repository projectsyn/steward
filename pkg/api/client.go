package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
)

// Client for the SYN API
type Client struct {
	BaseURL *url.URL
	Token   string

	httpClient *http.Client
}

// NewClient creates a default API client
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
		httpClient.Timeout = 10 * time.Second
	}
	return &Client{httpClient: httpClient}
}

type registerClusterRequest struct {
	Token        string `json:"token,omitempty"`
	Distribution string `json:"distribution,omitempty"`
	CloudType    string `json:"cloud_type,omitempty"`
	CloudRegion  string `json:"cloud_region,omitempty"`
}

type registerClusterResponse struct {
	Git *GitInfo `json:"git"`
}

// GitInfo represents information about a git repository
type GitInfo struct {
	HostName string `json:"host_name"`
	RepoName string `json:"repo_name"`
}

// RegisterCluster registers a new cluster to the SYN API
func (c *Client) RegisterCluster(ctx context.Context, cloudType, cloudRegion, distribution string) (*GitInfo, error) {

	cluster := registerClusterRequest{
		Token:        c.Token,
		CloudType:    cloudType,
		CloudRegion:  cloudRegion,
		Distribution: distribution,
	}
	req, err := c.newRequest(ctx, "POST", "/clusters/register", cluster)
	if err != nil {
		return nil, err
	}
	resp := registerClusterResponse{}
	log.Debugf("Make request to %s", req.URL.String())
	_, err = c.do(req, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Git, nil
}

func (c *Client) do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return resp, fmt.Errorf("Error occured, status %d", resp.StatusCode)
	}
	err = json.NewDecoder(resp.Body).Decode(v)
	return resp, err
}

func (c *Client) newRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	rel := &url.URL{Path: path}
	u := c.BaseURL.ResolveReference(rel)
	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, u.String(), buf)
	req = req.WithContext(ctx)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	return req, nil
}
