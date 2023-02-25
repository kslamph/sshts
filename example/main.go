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
	//address and port of ssh server to connect to
	sshAddress := "18.162.155.249:22"

	//address and listening port of socks5 server to start
	socks5Address := "localhost:1080"

	//username, private key path, and address of ssh server to connect to
	sshC, err := sshts.New("ec2-user", "/home/k/.ssh/id_rsa", sshAddress)
	if err != nil {
		log.Fatal(err)
	}
	err = sshC.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer sshC.Close()

	go sshC.StartSocks5Server(socks5Address)

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	fmt.Println("exiting")

}
