// Package komodo is a minimal client for the read API of Komodo Core
// (https://github.com/moghtech/komodo). It decodes only the fields the
// exporter uses; their names and values follow Komodo v2.3.
package komodo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client calls Komodo's read API with an API key.
type Client struct {
	url    string
	key    string
	secret string
	http   *http.Client
}

// NewClient returns a client for the Komodo Core at url, such as
// http://komodo:9120. The key needs read access to the resources to export.
func NewClient(url, key, secret string, httpClient *http.Client) *Client {
	return &Client{url: strings.TrimRight(url, "/"), key: key, secret: secret, http: httpClient}
}

// listAll asks a List request for every result instead of the first page.
var listAll = map[string]any{"limit": 0}

// read posts params to /read/<request> and decodes the response into out.
func (c *Client) read(ctx context.Context, request string, params, out any) error {
	body, err := json.Marshal(params)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/read/"+request, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.key)
	req.Header.Set("X-Api-Secret", c.secret)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", request, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s: %s: %s", request, resp.Status, bytes.TrimSpace(msg))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%s: %w", request, err)
	}
	return nil
}

// ListServers returns every server.
func (c *Client) ListServers(ctx context.Context) ([]Server, error) {
	var out []Server
	return out, c.read(ctx, "ListServers", listAll, &out)
}

// GetSystemStats returns the latest system stats of a server, by ID or name.
// Komodo serves them from a cache and fails for a disconnected server.
func (c *Client) GetSystemStats(ctx context.Context, server string) (SystemStats, error) {
	var out SystemStats
	return out, c.read(ctx, "GetSystemStats", map[string]string{"server": server}, &out)
}

// ListStacks returns every stack.
func (c *Client) ListStacks(ctx context.Context) ([]Stack, error) {
	var out []Stack
	return out, c.read(ctx, "ListStacks", listAll, &out)
}

// ListAllContainers returns the containers on every server.
func (c *Client) ListAllContainers(ctx context.Context) ([]Container, error) {
	var out []Container
	return out, c.read(ctx, "ListAllContainers", listAll, &out)
}

// ListResourceSyncs returns every resource sync.
func (c *Client) ListResourceSyncs(ctx context.Context) ([]Resource, error) {
	var out []Resource
	return out, c.read(ctx, "ListResourceSyncs", listAll, &out)
}

// ListProcedures returns every procedure.
func (c *Client) ListProcedures(ctx context.Context) ([]Resource, error) {
	var out []Resource
	return out, c.read(ctx, "ListProcedures", listAll, &out)
}

// ListRepos returns every repo.
func (c *Client) ListRepos(ctx context.Context) ([]Resource, error) {
	var out []Resource
	return out, c.read(ctx, "ListRepos", listAll, &out)
}
