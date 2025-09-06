package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kslamph/sshts"
)

func main() {
	// Example of using host key verification with known_hosts file
	secureTunnelExample()
}

func secureTunnelExample() {
	// Address and port of ssh server to connect to
	sshAddress := "example.com:22"

	// Local port to listen on
	localTunnel := "localhost:8080"

	// Server to tunnel to on remote, usually localhost but can be any address
	remoteTunnel := "localhost:80"

	// Using known_hosts file verification (recommended)
	// This will fail if the host key is not in the known_hosts file
	sshC, err := sshts.NewWithKnownHosts("username", "/path/to/private/key", sshAddress, "/path/to/known_hosts")
	if err != nil {
		log.Printf("Failed to create SSH connection: %v", err)
		log.Println("Make sure you have the correct paths and that the host key is in your known_hosts file")
		return
	}

	err = sshC.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer sshC.Close()

	go func() {
		err := sshC.StartTunnel(localTunnel, remoteTunnel)
		if err != nil {
			log.Printf("Tunnel stopped with error: %v", err)
		}
	}()

	fmt.Println("Secure SSH tunnel started")
	fmt.Println("Press ctrl+c to exit")
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("Exiting")
}
