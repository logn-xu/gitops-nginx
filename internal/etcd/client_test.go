package etcd

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/logn-xu/gitops-nginx/internal/config"
)

func getTestConfig() config.EtcdConfig {
	endpoints := []string{"localhost:2379"}
	if envEndpoints := os.Getenv("ETCD_ENDPOINTS"); envEndpoints != "" {
		endpoints = strings.Split(envEndpoints, ",")
	}
	return config.EtcdConfig{
		Endpoints: endpoints,
	}
}

func TestNewClient(t *testing.T) {
	cfg := getTestConfig()

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if client == nil {
		t.Fatal("NewClient returned nil client")
	}
	client.Close()
}

func TestClient_Operations(t *testing.T) {
	cfg := getTestConfig()

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check connectivity before running tests
	// This allows the test to pass (skip) in environments without etcd
	_, err = client.Get(ctx, "health_check")
	if err != nil {
		t.Logf("Skipping integration test (etcd not reachable): %v", err)
		t.SkipNow()
	}

	key := "/test/gitops-nginx/key"
	val := "test_value"

	// Test Put
	t.Run("Put", func(t *testing.T) {
		_, err := client.Put(ctx, key, val)
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	})

	// Test Get
	t.Run("Get", func(t *testing.T) {
		resp, err := client.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if len(resp.Kvs) == 0 {
			t.Errorf("Expected key %s to exist", key)
		} else if string(resp.Kvs[0].Value) != val {
			t.Errorf("Expected value %s, got %s", val, string(resp.Kvs[0].Value))
		}
	})

	// Test GetPrefix
	t.Run("GetPrefix", func(t *testing.T) {
		prefixKey := "/test/gitops-nginx/prefix/1"
		prefixVal := "prefix_val_1"
		_, err = client.Put(ctx, prefixKey, prefixVal)
		if err != nil {
			t.Fatalf("Put prefix failed: %v", err)
		}
		defer client.Delete(context.Background(), prefixKey)

		prefixResp, err := client.GetPrefix(ctx, "/test/gitops-nginx/prefix")
		if err != nil {
			t.Fatalf("GetPrefix failed: %v", err)
		}
		if len(prefixResp.Kvs) == 0 {
			t.Errorf("Expected prefix keys to exist")
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		_, err := client.Delete(ctx, key)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify Delete
		resp, err := client.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get after delete failed: %v", err)
		}
		if len(resp.Kvs) > 0 {
			t.Errorf("Expected key %s to be deleted", key)
		}
	})
}
