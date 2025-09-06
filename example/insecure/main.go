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
	tunnelexample()

	// Example of using host key verification with known_hosts file
	// secureTunnelExample()
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
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("exiting")

}
