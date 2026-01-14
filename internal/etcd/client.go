package etcd

import (
	"context"
	"fmt"
	"time"

	"github.com/logn-xu/gitops-nginx/internal/config"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Client wraps the etcd v3 client.
type Client struct {
	*clientv3.Client
}

// NewClient creates a new etcd client.
func NewClient(cfg config.EtcdConfig) (*Client, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}
	return &Client{cli}, nil
}

// Put stores a key-value pair in etcd.
func (c *Client) Put(ctx context.Context, key, value string) (*clientv3.PutResponse, error) {
	return c.Client.Put(ctx, key, value)
}

// Get retrieves a value from etcd by key.
func (c *Client) Get(ctx context.Context, key string) (*clientv3.GetResponse, error) {
	return c.Client.Get(ctx, key)
}

// GetPrefix retrieves keys with a given prefix.
func (c *Client) GetPrefix(ctx context.Context, key string) (*clientv3.GetResponse, error) {
	return c.Client.Get(ctx, key, clientv3.WithPrefix())
}

// Delete removes a key from etcd.
func (c *Client) Delete(ctx context.Context, key string) (*clientv3.DeleteResponse, error) {
	return c.Client.Delete(ctx, key)
}
