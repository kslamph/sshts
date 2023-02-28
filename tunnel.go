package sshts

import (
	"fmt"
	"io"
	"net"
)

//StartTunnel listne a local port and map to remote

func (s *SSHConn) StartTunnel(local, remote string) error {
	listener, err := net.Listen("tcp", local)
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go s.forward(conn, remote)
	}
}
func (s *SSHConn) forward(localConn net.Conn, remote string) {

	remoteConn, err := s.sshClient.Dial("tcp", remote)
	if err != nil {
		fmt.Printf("remote dial error: %s", err)
		return
	}

	copyConn := func(writer, reader net.Conn) {
		_, err := io.Copy(writer, reader)
		if err != nil {
			fmt.Printf("io.Copy error: %s", err)
		}
	}
	go copyConn(localConn, remoteConn)
	go copyConn(remoteConn, localConn)
}
