# github.com/kslamph/sshts

This package enables users to:

- Establish a tunnel to a remote server via SSH
- Use the remote server as a SOCKS5 proxy via SSH


## Example:
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

	sshC, err := sshts.New("root", "/home/k/.ssh/id_rsa", sshAddress)
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
	sshC, err := sshts.New("proxy", "/home/k/.ssh/proxy.key", sshAddress)

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
