package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/kslamph/sshts"
)

func main() {
	// Example of using the new simplified tunnel API with secure host key verification

	// Read private key
	key, err := os.ReadFile("/path/to/private/key")
	if err != nil {
		log.Fatal("Failed to read private key:", err)
	}

	// Parse private key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatal("Failed to parse private key:", err)
	}

	// Load host key verification from known_hosts file
	hostKeyCallback, err := knownhosts.New("/path/to/known_hosts")
	if err != nil {
		log.Fatal("Failed to load known_hosts file:", err)
	}
	
	// Create tunnel configuration
	config := &sshts.TunnelConfig{
		User: "username",
		AuthMethods: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback, // Secure host key verification
		// HostKeyCallback: ssh.InsecureIgnoreHostKey(), // INSECURE: Alternative to disable host key verification
		SSHTimeout:      30 * time.Second,
		DialTimeout:     10 * time.Second,
		MaxConnections:  50,
		BufferSize:      64 * 1024, // 64KB
	}
	
	// Note: For production use, host key verification is strongly recommended
	// If you want to disable host key verification (NOT recommended), uncomment the line above
	// Warning: Disabling host key verification makes connections vulnerable to man-in-the-middle attacks

	// Create tunnel - tunnel is NOT yet usable, no SSH connection established
	tunnel := sshts.NewTunnel(
		"localhost:8080", // Local address to listen on
		"localhost:80",   // Remote address to forward to
		config,
	)

	// Create context for the tunnel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start tunnel - tunnel becomes usable after this call
	// Returns a cancel function to stop the tunnel and an error if startup fails
	tunnelCancel, err := tunnel.Start(ctx, "ssh-server.example.com:22")
	if err != nil {
		log.Fatal("Failed to start tunnel:", err)
	}

	// TUNNEL IS NOW USABLE - connections will be forwarded through SSH tunnel
	fmt.Println("Secure tunnel started successfully")
	fmt.Println("Tunnel is now ready to forward connections")

	// In a real application, you would typically:
	// 1. Run your application logic here
	// 2. Call tunnelCancel() and tunnel.Close() when shutting down

	// For this example, we'll just sleep to simulate a running service
	// In practice, you'd handle graceful shutdown with signal handling or other mechanisms
	time.Sleep(10 * time.Second)

	// Stop tunnel - after this call, tunnel is NO LONGER USABLE
	tunnelCancel()
	tunnel.Close()

	// TUNNEL IS NOW UNUSABLE - all resources cleaned up
	fmt.Println("Tunnel stopped")
}
