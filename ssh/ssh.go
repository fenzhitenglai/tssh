package ssh

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"time"

	"github.com/containerd/console"
	"golang.org/x/crypto/ssh"
)

func Connect(ip string, port int, cfg *ssh.ClientConfig) (*ssh.Client, error) {
	addr := fmt.Sprintf("%s:%d", ip, port)
	sshClient, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, err
	}
	return sshClient, nil
}

func RunTerminal(c *ssh.Client, in io.Reader, stdOut, stdErr io.Writer) error {
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	session.Signal(ssh.SIGINT)
	session.Stdout = stdOut
	session.Stderr = stdErr
	session.Stdin = in
	var (
		current = console.Current()
		ws      console.WinSize
	)
	defer current.Reset()

	if err = current.SetRaw(); err != nil {
		return err
	}

	if ws, err = current.Size(); err != nil {
		return err
	}

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     //打开回显
		ssh.TTY_OP_ISPEED: 14400, //输入速率 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, //输出速率 14.4kbaud
		ssh.VSTATUS:       1,
	}

	//Request pseudo terminal
	if err = session.RequestPty("xterm-256color", int(ws.Height), int(ws.Width), modes); err != nil {
		return err
	}

	if err = session.Shell(); err != nil {
		return err
	}
	return session.Wait()
}

func PwdCfg(user, pwd string) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.Password(pwd)},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: 10 * time.Second,
	}
}

func PkCfg(user, pkPath string) (*ssh.ClientConfig, error) {
	pemBytes, err := ioutil.ReadFile(pkPath)
	if err != nil {
		return nil, fmt.Errorf("Reading private key file failed %v", err)
	}

	signer, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		return nil, fmt.Errorf("Parsing plain private key failed %v", err)
	}
	return &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: 10 * time.Second,
	}, nil
}
