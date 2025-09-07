# github.com/kslamph/sshts

This package enables users to:

- Establish a tunnel to a remote server via SSH
- Verify SSH host keys for improved security
- Graceful shutdown with context cancellation
- Configurable connection limits and timeouts
- Improved performance with buffered data transfer

## Usage

The package provides a simple API for creating SSH tunnels:

```go
// Create tunnel configuration
config := &sshts.TunnelConfig{
    User: "username",
    AuthMethods: []ssh.AuthMethod{
        ssh.PublicKeys(signer),
    },
    HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Or use knownhosts for security
}

// Create tunnel
tunnel := sshts.NewTunnel(
    "localhost:8080",    // Local address to listen on
    "localhost:80",      // Remote address to forward to
    config,
)

// Start tunnel with context
ctx, cancel := context.WithCancel(context.Background())
tunnelCancel, err := tunnel.Start(ctx, "ssh-server.example.com:22")
if err != nil {
    log.Fatal("Failed to start tunnel:", err)
}

// To stop the tunnel
tunnelCancel()
tunnel.Close()
```

## Host Key Verification

The package supports proper SSH host key verification for security:

1. **Insecure method (default)**: Uses `ssh.InsecureIgnoreHostKey()` - not recommended for production
2. **Known hosts file**: Verifies host keys against a `known_hosts` file (recommended)

Example using known hosts verification:
```go
hostKeyCallback, err := knownhosts.New("/path/to/known_hosts")
if err != nil {
    log.Fatal("Failed to load known_hosts file:", err)
}

config := &sshts.TunnelConfig{
    User: "username",
    AuthMethods: []ssh.AuthMethod{
        ssh.PublicKeys(signer),
    },
    HostKeyCallback: hostKeyCallback, // Secure host key verification
}
```

To set up host key verification:
1. Create a `known_hosts` file (usually located at `~/.ssh/known_hosts`)
2. Add the remote server's host key to this file
3. Use `knownhosts.New()` to create a host key callback

You can manually add entries to known_hosts or use ssh-keyscan:
```bash
ssh-keyscan server.example.com >> ~/.ssh/known_hosts
```

## Features

### Graceful Shutdown
The package supports context cancellation for graceful shutdowns. When the connection is closed, all goroutines are properly cancelled and resources are cleaned up.

### Connection Limits and Timeouts
You can configure connection limits and timeouts to prevent resource exhaustion:

```go
config := &sshts.TunnelConfig{
    // ... other config
    MaxConnections: 100,
    DialTimeout: 30 * time.Second,
    SSHTimeout: 30 * time.Second,
}
```

### Performance
Data transfer uses `io.CopyBuffer` with a shared buffer pool for better performance:

```go
config := &sshts.TunnelConfig{
    // ... other config
    BufferSize: 64 * 1024, // 64KB buffers
}
```