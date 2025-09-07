package sshts

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"
)

// Tunnel represents an SSH tunnel
type Tunnel struct {
	// Configuration
	localAddr  string
	remoteAddr string
	sshConfig  *ssh.ClientConfig
	
	// State
	client   *ssh.Client
	listener net.Listener
	
	// Concurrency control
	ctx        context.Context
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
	
	// Performance
	bufferPool *bufferPool
	
	// Limits
	maxConnections int64
	connCount      int64
	
	// Timeouts
	dialTimeout time.Duration
}

// TunnelConfig holds tunnel configuration
type TunnelConfig struct {
	// SSH configuration
	User        string
	AuthMethods []ssh.AuthMethod
	HostKeyCallback ssh.HostKeyCallback
	SSHTimeout  time.Duration
	
	// Tunnel configuration
	DialTimeout    time.Duration
	MaxConnections int
	BufferSize     int
}

// NewTunnel creates a new SSH tunnel
func NewTunnel(localAddr, remoteAddr string, config *TunnelConfig) *Tunnel {
	// Set default values
	if config.HostKeyCallback == nil {
		config.HostKeyCallback = ssh.InsecureIgnoreHostKey() // Not recommended for production
	}
	if config.SSHTimeout == 0 {
		config.SSHTimeout = 30 * time.Second
	}
	if config.DialTimeout == 0 {
		config.DialTimeout = 30 * time.Second
	}
	if config.MaxConnections == 0 {
		config.MaxConnections = 100
	}
	if config.BufferSize == 0 {
		config.BufferSize = 32 * 1024 // 32KB
	}
	
	// Create SSH client config
	sshConfig := &ssh.ClientConfig{
		User:            config.User,
		Auth:            config.AuthMethods,
		HostKeyCallback: config.HostKeyCallback,
		Timeout:         config.SSHTimeout,
	}
	
	return &Tunnel{
		localAddr:      localAddr,
		remoteAddr:     remoteAddr,
		sshConfig:      sshConfig,
		bufferPool:     newBufferPool(config.BufferSize),
		maxConnections: int64(config.MaxConnections),
		dialTimeout:    config.DialTimeout,
	}
}

// Start starts the tunnel with the provided context
// Returns a cancel function and an error
// Usage: cancel, err := tunnel.Start(ctx)
func (t *Tunnel) Start(ctx context.Context, sshServerAddr string) (context.CancelFunc, error) {
	// Connect to SSH server
	client, err := ssh.Dial("tcp", sshServerAddr, t.sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH server: %w", err)
	}
	t.client = client
	
	// Create listener for local address
	listener, err := net.Listen("tcp", t.localAddr)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to listen on %s: %w", t.localAddr, err)
	}
	t.listener = listener
	
	// Create context with cancel for tunnel management
	tunnelCtx, cancel := context.WithCancel(ctx)
	t.ctx = tunnelCtx
	t.cancelFunc = cancel
	
	// Start accepting connections
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.acceptLoop()
	}()
	
	return cancel, nil
}

// acceptLoop accepts incoming connections and forwards them
func (t *Tunnel) acceptLoop() {
	defer t.listener.Close()
	
	for {
		select {
		case <-t.ctx.Done():
			return
		default:
		}
		
		// Accept new connection
		conn, err := t.listener.Accept()
		if err != nil {
			select {
			case <-t.ctx.Done():
				return
			default:
				// Log error and continue
				continue
			}
		}
		
		// Check connection limits
		if t.maxConnections > 0 {
			current := atomic.AddInt64(&t.connCount, 1)
			defer atomic.AddInt64(&t.connCount, -1)
			
			if current > t.maxConnections {
				conn.Close()
				// Log error: connection limit exceeded
				continue
			}
		}
		
		// Handle connection in goroutine
		t.wg.Add(1)
		go func(c net.Conn) {
			defer t.wg.Done()
			t.handleConnection(c)
		}(conn)
	}
}

// handleConnection handles a single connection
func (t *Tunnel) handleConnection(localConn net.Conn) {
	defer localConn.Close()
	
	// Create context with timeout for dialing
	dialCtx, cancel := context.WithTimeout(t.ctx, t.dialTimeout)
	defer cancel()
	
	// Connect to remote address through SSH
	type dialResult struct {
		conn net.Conn
		err  error
	}
	
	resultChan := make(chan dialResult, 1)
	go func() {
		sshConn, err := t.client.Dial("tcp", t.remoteAddr)
		resultChan <- dialResult{conn: sshConn, err: err}
	}()
	
	var remoteConn net.Conn
	select {
	case result := <-resultChan:
		if result.err != nil {
			// Log error
			return
		}
		remoteConn = result.conn
	case <-dialCtx.Done():
		// Log timeout error
		return
	}
	
	defer remoteConn.Close()
	
	// Forward data between connections
	t.forwardData(localConn, remoteConn)
}

// forwardData forwards data between two connections
func (t *Tunnel) forwardData(conn1, conn2 net.Conn) {
	// Get buffer from pool
	buf := t.bufferPool.Get()
	defer t.bufferPool.Put(buf)
	
	// Create context for this forwarding operation
	forwardCtx, cancel := context.WithCancel(t.ctx)
	defer cancel()
	
	// Forward data in both directions
	var wg sync.WaitGroup
	wg.Add(2)
	
	// conn1 -> conn2
	go func() {
		defer wg.Done()
		t.copyData(conn1, conn2, buf, forwardCtx)
	}()
	
	// conn2 -> conn1
	go func() {
		defer wg.Done()
		t.copyData(conn2, conn1, buf, forwardCtx)
	}()
	
	// Wait for either context cancellation or data transfer completion
	go func() {
		<-forwardCtx.Done()
		conn1.Close()
		conn2.Close()
	}()
	
	wg.Wait()
}

// copyData copies data from src to dst using the provided buffer
func (t *Tunnel) copyData(src, dst net.Conn, buf []byte, ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		
		// Use io.CopyBuffer for efficient data transfer
		_, err := io.CopyBuffer(dst, src, buf)
		if err != nil {
			return
		}
	}
}

// Close stops the tunnel and closes all connections
func (t *Tunnel) Close() error {
	if t.cancelFunc != nil {
		t.cancelFunc()
	}
	
	if t.listener != nil {
		t.listener.Close()
	}
	
	if t.client != nil {
		t.client.Close()
	}
	
	// Wait for all goroutines to finish
	t.wg.Wait()
	
	return nil
}

// bufferPool is a pool of byte buffers for efficient memory usage
type bufferPool struct {
	pool sync.Pool
}

// newBufferPool creates a new buffer pool with the specified buffer size
func newBufferPool(bufferSize int) *bufferPool {
	return &bufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, bufferSize)
			},
		},
	}
}

// Get returns a buffer from the pool
func (bp *bufferPool) Get() []byte {
	return bp.pool.Get().([]byte)
}

// Put returns a buffer to the pool
func (bp *bufferPool) Put(buf []byte) {
	bp.pool.Put(buf)
}