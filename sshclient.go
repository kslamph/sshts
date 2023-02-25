package sshts

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

type SSHConn struct {
	sshConf    *ssh.ClientConfig
	sshClient  *ssh.Client
	serverAddr string
	status     int64
}

// New("user", "/home/user/.ssh/id_rsa", "1.1.1.1:22")
func New(user, rsaKeyfile, serverAddr string) (*SSHConn, error) {

	key, err := os.ReadFile(rsaKeyfile)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key: %v", err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %v", err)
	}

	sshConf := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return &SSHConn{
		sshConf:    sshConf,
		serverAddr: serverAddr,
		status:     0,
		sshClient:  nil,
	}, nil
}

func (s *SSHConn) Connect() error {
	client, err := ssh.Dial("tcp", s.serverAddr, s.sshConf)
	if err != nil {
		return fmt.Errorf("error connect to ssh server: %v", err)
	}
	s.sshClient = client
	s.status = 1
	return nil
}

func (s *SSHConn) GetStatus() int64 {
	return s.status
}

func (s *SSHConn) Close() error {
	if s.sshClient != nil {
		err := s.sshClient.Close()
		if err != nil {
			return fmt.Errorf("error close ssh connection: %v", err)
		}
	}
	s.status = 0
	return nil
}
