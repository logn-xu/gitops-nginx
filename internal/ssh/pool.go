package ssh

import (
	"sync"

	"github.com/logn-xu/gitops-nginx/internal/config"
	"github.com/logn-xu/gitops-nginx/pkg/log"
)

const defaultMaxCapacity = 10

type SFTPPool struct {
	pool        chan *Client                                             // 使用带缓冲的 channel 存储连接
	factory     func(serverConfig *config.ServerConfig) (*Client, error) // 创建连接的工厂函数
	maxCapacity int                                                      // 最大连接数
	mu          sync.Mutex                                               // 保护 currentSize (如果需要精确控制创建数量)
}

// NewSFTPPool
func NewSFTPPool(serverConfig *config.ServerConfig, maxCap int) (*SFTPPool, error) {
	if maxCap == 0 {
		maxCap = defaultMaxCapacity
	}

	factory := func(serverConfig *config.ServerConfig) (*Client, error) {
		return NewClient(serverConfig)
	}

	// 2. create pool object
	p := &SFTPPool{
		pool:        make(chan *Client, maxCap),
		factory:     factory,
		maxCapacity: maxCap,
	}

	// Pre-warm the pool
	for i := 0; i < maxCap/2; i++ {
		client, err := p.factory(serverConfig)
		if err != nil {
			return nil, err
		}
		p.pool <- client
	}

	return p, nil
}

// Get live conn
func (p *SFTPPool) Get(serverConfig *config.ServerConfig) (*Client, error) {
	select {
	case client := <-p.pool:
		// test conn is active
		_, err := client.sftpClient.Getwd()
		if err != nil {
			log.Logger.WithField("sftp_pool", serverConfig.Host).
				Warn("conn is not active,drop this conn")
			client.Close()
			return p.factory(serverConfig) // 递归或直接调用工厂新建
		}
		log.Logger.WithField("sftp_pool", serverConfig.Host).
			Info("Reuse existing connection")
		return client, nil
	default:
		// Pool is empty, create a new one.
		// Note: This is a simplified implementation.
		// While strict concurrency limits usually require semaphores,
		// "create if empty, destroy if full on return" is simple and effective.
		log.Logger.WithField("sftp_pool", serverConfig.Host).Info("Pool empty, creating new connection")
		return p.factory(serverConfig)
	}
}

// Put returns a connection to the pool.
func (p *SFTPPool) Put(client *Client) {
	// Optional: health check before returning
	select {
	case p.pool <- client:
		// Successfully returned
		log.Logger.Info("Connection returned to pool")
	default:
		// Pool is full (e.g., we created too many new connections when the pool was empty)
		// Close the redundant connection.
		log.Logger.Info("Pool full, destroying redundant connection")
		client.Close()
	}
}
