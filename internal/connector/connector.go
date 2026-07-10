package connector

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/gituser/redis-analyze/internal/models"
)

// Connect creates a redis.UniversalClient based on the provided options.
// Returns the client, a cleanup function, and any error encountered.
func Connect(ctx context.Context, opts models.ScanOptions) (redis.UniversalClient, func(), error) {
	var client redis.UniversalClient

	tlsConfig := buildTLSConfig(opts.TLS)

	switch opts.Mode {
	case "cluster":
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        opts.Addrs,
			Username:     opts.Username,
			Password:     opts.Password,
			TLSConfig:    tlsConfig,
			DialTimeout:  time.Duration(opts.Timeout) * time.Second,
			ReadTimeout:  time.Duration(opts.Timeout) * time.Second,
			WriteTimeout: time.Duration(opts.Timeout) * time.Second,
		})

	case "sentinel":
		client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:         opts.MasterName,
			SentinelAddrs:      opts.Addrs,
			Username:           opts.Username,
			Password:           opts.Password,
			SentinelPassword:   opts.SentinelPass,
			DB:                 opts.DB,
			TLSConfig:          tlsConfig,
			DialTimeout:        time.Duration(opts.Timeout) * time.Second,
			ReadTimeout:        time.Duration(opts.Timeout) * time.Second,
			WriteTimeout:       time.Duration(opts.Timeout) * time.Second,
		})

	default: // standalone
		addr := fmt.Sprintf("%s:%d", opts.Host, opts.Port)
		if len(opts.Addrs) > 0 && opts.Addrs[0] != "" {
			addr = opts.Addrs[0]
		}
		client = redis.NewClient(&redis.Options{
			Addr:         addr,
			Username:     opts.Username,
			Password:     opts.Password,
			DB:           opts.DB,
			TLSConfig:    tlsConfig,
			DialTimeout:  time.Duration(opts.Timeout) * time.Second,
			ReadTimeout:  time.Duration(opts.Timeout) * time.Second,
			WriteTimeout: time.Duration(opts.Timeout) * time.Second,
		})
	}

	// Verify connection
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("failed to connect to Redis (%s mode): %w", opts.Mode, err)
	}

	cleanup := func() {
		_ = client.Close()
	}

	return client, cleanup, nil
}

// FetchServerInfo collects basic server info for the report.
func FetchServerInfo(ctx context.Context, client redis.UniversalClient, opts models.ScanOptions) models.ServerInfo {
	info := models.ServerInfo{
		Host: opts.Host,
		Port: opts.Port,
		DB:   opts.DB,
		Mode: opts.Mode,
	}

	infoStr, err := client.Info(ctx, "server").Result()
	if err == nil {
		info.Version = parseInfoValue(infoStr, "redis_version")
		info.Uptime = parseIntInfo(infoStr, "uptime_in_seconds")
		info.Os = parseInfoValue(infoStr, "os")
	}

	memStr, err := client.Info(ctx, "memory").Result()
	if err == nil {
		info.UsedMem = parseIntInfo(memStr, "used_memory")
	}

	// Try to get key count from the current database
	keyspace, err := client.Info(ctx, "keyspace").Result()
	if err == nil {
		// For standalone: find db0 keys
		dbKey := fmt.Sprintf("db%d", opts.DB)
		val := parseInfoValue(keyspace, dbKey)
		// Format: keys=12345,expires=...
		if val != "" {
			parts := strings.Split(val, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if strings.HasPrefix(p, "keys=") {
					var k int64
					_, _ = fmt.Sscanf(p, "keys=%d", &k)
					info.KeysCount = k
				}
			}
		}
	}

	return info
}

// FetchClusterNodeCount returns the number of master nodes in a cluster.
// For standalone/sentinel, returns 1.
func FetchClusterNodeCount(ctx context.Context, client redis.UniversalClient) int {
	if clusterClient, ok := client.(*redis.ClusterClient); ok {
		nodes, err := clusterClient.ClusterNodes(ctx).Result()
		if err != nil {
			return 0
		}
		count := 0
		for _, line := range strings.Split(nodes, "\n") {
			if strings.Contains(line, "master") {
				count++
			}
		}
		return count
	}
	return 1
}

func buildTLSConfig(enabled bool) *tls.Config {
	if !enabled {
		return nil
	}
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
}

func parseInfoValue(info, key string) string {
	prefix := key + ":"
	for _, line := range strings.Split(info, "\r\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func parseIntInfo(info, key string) int64 {
	val := parseInfoValue(info, key)
	if val == "" {
		return 0
	}
	var n int64
	_, _ = fmt.Sscanf(val, "%d", &n)
	return n
}
