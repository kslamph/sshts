# github.com/kslamph/sshts

This package enables users to:

- Establish a tunnel to a remote server via SSH
- Use the remote server as a SOCKS5 proxy via SSH
- Verify SSH host keys for improved security

## Host Key Verification

The package now supports proper SSH host key verification, which is crucial for security. There are three ways to verify host keys:

1. **Insecure method (default)**: Uses `ssh.InsecureIgnoreHostKey()` - not recommended for production
2. **Known hosts file**: Verifies host keys against a `known_hosts` file (recommended)
3. **Custom callback**: Provide your own `ssh.HostKeyCallback` function

Example using known hosts verification:
```go
sshC, err := sshts.NewWithKnownHosts("username", "/path/to/private/key", "server:port", "/path/to/known_hosts")
```

To set up host key verification:

1. Create a `known_hosts` file (usually located at `~/.ssh/known_hosts`)
2. Add the remote server's host key to this file
3. Use `NewWithKnownHosts()` instead of `New()`

You can manually add entries to known_hosts or use ssh-keyscan:
```bash
ssh-keyscan server.example.com >> ~/.ssh/known_hosts
```
Here's an example of how to use this package:
```go
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"github.com/kslamph/sshts"
)

func main() {
	tunnelexample()
}

/* tunnel exmaple:
there is a remote server 101.32.14.95 , we have ssh access to it;
there is a postgres sql server running on it and listen on localhost:5432;
and this 101.32.14.95:5432 is not accessible from outside.

we want to access the postgres server by localhost:15432

this setup increased the security of the postgres server, because it is not accessible from outside, and we can only access it by ssh tunneling.
there are 3 advantages of this setup:
using key to access ssh server is a proven safe way to access remote server
ssh tunneling is encrypted, so the data is safe at transport layer
we dont have to expose the postgres server interfaces other than localhost, so it is more secure
*/

func tunnelexample() {
	//address and port of ssh server to connect to
	sshAddress := "101.32.14.95:22"

	//local port to listen on
	localTunnel := "localhost:15432"

	//server to tunnel to on remote, usually localhost but can be any address
	remoteTunnel := "localhost:5432"

	// Method 1: Using the default insecure method (not recommended for production)
	// sshC, err := sshts.New("root", "/home/k/.ssh/id_rsa", sshAddress)
	
	// Method 2: Using a custom host key callback
	// hostKeyCallback := ssh.FixedHostKey(theHostKey) // where theHostKey is of type ssh.PublicKey
	// sshC, err := sshts.NewWithHostKeyCallback("root", "/home/k/.ssh/id_rsa", sshAddress, hostKeyCallback)
	
	// Method 3: Using known_hosts file verification (recommended)
	sshC, err := sshts.NewWithKnownHosts("root", "/home/k/.ssh/id_rsa", sshAddress, "/home/k/.ssh/known_hosts")
	if err != nil {
		log.Fatal(err)
	}
	err = sshC.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer sshC.Close()

	go sshC.StartTunnel(localTunnel, remoteTunnel)
	fmt.Println("press ctrl+c to exit")
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("exiting")

}

/* socks5 exmaple:
there is a remote server 101.32.14.95 , we have ssh access to it, we want use it as a socks5 proxy
and let the socks5 proxy listen on localhost:1080

this setup has some advantages over proxy:
1. it is encrypted, so the data is safe at transport layer
2. there is no proxy server for maintain and it is easy to setup
*/

func socks5example() {
	//address and port of ssh server to connect to
	sshAddress := "18.162.151.198:22"

	//address and listening port of socks5 server to start
	socks5Address := "localhost:1080"

	//username, private key path, and address of ssh server to connect to
	// Using known_hosts file verification (recommended)
	sshC, err := sshts.NewWithKnownHosts("proxy", "/home/k/.ssh/proxy.key", sshAddress, "/home/k/.ssh/known_hosts")
	if err != nil {
		log.Fatal(err)
	}
	err = sshC.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer sshC.Close()

	go sshC.StartSocks5Server(socks5Address)
	for sshC.GetStatus() < 2 {
	}

	httpget()

	fmt.Println("press ctrl+c to exit")
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("exiting")

}

func httpget() {
	proxyURL, err := url.Parse("socks5://localhost:1080")
	if err != nil {
		log.Fatal(err)
	}
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{Transport: transport}
	resp, err := client.Get("http://ifconfig.me")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(body))
}
```
