package sshts

import (
	"context"
	"fmt"
	"net"

	"github.com/armon/go-socks5"
)

func (s *SSHConn) StartSocks5Server(socks5Address string) error {
	if s.sshClient == nil || s.status == 0 {
		return fmt.Errorf("ssh client is not connected")
	}
	conf := &socks5.Config{
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return s.sshClient.Dial(network, addr)
		},
	}

	serverSocks, err := socks5.New(conf)

	if err != nil {
		return fmt.Errorf("failed to create socks5 server %v", err)
	}

	l, err := net.Listen("tcp", socks5Address)

	if err != nil {
		return fmt.Errorf("failed to listen socks5 server on %s, %v", socks5Address, err)
	}
	s.status = 2

	if err := serverSocks.Serve(l); err != nil {
		return fmt.Errorf("failed to start socks5 server %v", err)
	}
	return nil
}
