package ssh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
	"golang.org/x/term"
)

// Client client
type Client struct {
	opts   Options
	client *ssh.Client
	pass   string
}

// New new service
func New(opts ...Option) *Client {
	options := newOptions(opts...)
	client := Client{
		opts: options,
	}
	return &client
}

// Init initialises options.
func (c *Client) Init(opts ...Option) {
	// process options
	for _, o := range opts {
		o(&c.opts)
	}
}

// LoadConfig load config
func (c *Client) LoadConfig(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &c.opts)
}

// FastConnect connect2
func (c *Client) FastConnect() error {
	group, _ := errgroup.WithContext(context.Background())
	var cerr error
	var once sync.Once
	var passwords []string
	if c.opts.Password != "" {
		passwords = append(passwords, c.opts.Password)
	}
	passwords = append(passwords, c.opts.Passwords...)

	for _, passwd := range passwords {
		password := passwd
		group.Go(func() error {
			client, err := c.connect(c.opts.Username, ssh.Password(password), PasswordKeyboardInteractive(password))
			if err != nil {
				once.Do(func() { cerr = err })
				return nil
			}
			c.client = client
			c.pass = password
			return io.EOF
		})
	}
	if err := group.Wait(); err != nil && errors.Is(err, io.EOF) {
		return nil
	}
	return cerr
}

// Connect dail connect
func (c *Client) Connect() error {

	auths, err := c.authMethods()
	if err != nil {
		return fmt.Errorf("auth: %v", err)
	}
	c.client, err = c.connect(c.opts.Username, auths...)
	return err
}

func (c *Client) connect(user string, auths ...ssh.AuthMethod) (*ssh.Client, error) {
	sshConfig := ssh.Config{}
	sshConfig.SetDefaults()
	sshConfig.KeyExchanges = append(
		sshConfig.KeyExchanges,
		"diffie-hellman-group-exchange-sha256",
		"diffie-hellman-group-exchange-sha1",
	)
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            auths,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(c.opts.Timeout) * time.Second,
		Config:          sshConfig,
	}

	addr := fmt.Sprintf("%s:%d", c.opts.IP, c.opts.Port)
	return ssh.Dial("tcp", addr, config)
}

// CmdOutBytes cmd out bytes
func (c *Client) CmdOutBytes(cmd string) ([]byte, error) {
	sess, err := c.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("create ssh session: %v", err)
	}
	defer sess.Close()
	for k, v := range c.opts.Env {
		if err := sess.Setenv(k, v); err != nil {
			return nil, fmt.Errorf("Setenv: %v", err)
		}
	}

	if c.opts.Pseudo {
		// Set up terminal modes
		modes := ssh.TerminalModes{
			ssh.ECHO:          1, //是否回显输入的命令
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}
		// Request pseudo terminal
		if err = sess.RequestPty("xterm-256color", 0, 0, modes); err != nil {
			return nil, fmt.Errorf("request pty: %v", err)
		}
	}
	return sess.CombinedOutput(cmd)
}

// Upload upload
func (c *Client) Upload(src, dest string, mode os.FileMode) error {
	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return fmt.Errorf("建立sftp出错: %v", err)
	}
	defer sftpClient.Close()

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("读取本地文件[%s]出错: %v", src, err)
	}
	defer srcFile.Close()

	destFile, err := sftpClient.Create(dest)
	if err != nil {
		return fmt.Errorf("创建远程文件[%s]出错: %v", dest, err)
	}
	defer destFile.Close()

	size := 0
	buf := make([]byte, 1024*1024)
	for {
		n, err := srcFile.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("上传文件read出错#1: %v", err)
		}
		if n == 0 {
			break
		}
		if _, err := destFile.Write(buf[:n]); err != nil {
			return fmt.Errorf("上传文件write出错#2: %v", err)
		}
		size += n
	}
	return destFile.Chmod(mode)
}

// StartTerminal StartTerminal
func (c *Client) StartTerminal() error {
	sess, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("创建Session出错: %v", err)
	}
	defer sess.Close()

	for k, v := range c.opts.Env {
		if err := sess.Setenv(k, v); err != nil {
			return fmt.Errorf("Setenv[%s=%s]: %w", k, v, err)
		}
	}

	sess.Stdin = os.Stdin

	sess.Stdout = os.Stdout
	sess.Stderr = os.Stderr

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("创建文件描述符出错: %v", err)
	}
	defer term.Restore(fd, oldState) // nolint: errcheck

	width, height := 0, 0

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1, //是否回显输入的命令
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	// Request pseudo terminal
	if err = sess.RequestPty("xterm-256color", height, width, modes); err != nil {
		return fmt.Errorf("创建终端出错: %v", err)
	}
	// Set up terminal modes
	if err = sess.Shell(); err != nil {
		return fmt.Errorf("执行Shell出错: %v", err)
	}
	go func(fd int) error {
		t := time.NewTimer(time.Millisecond * 0)
		for {
			select {
			case <-t.C:

				width, height, err = term.GetSize(fd)
				if err != nil {
					return fmt.Errorf("获取窗口宽高出错: %v", err)
				}
				if err = sess.WindowChange(height, width); err != nil {
					return fmt.Errorf("改变窗口大小出错: %v", err)
				}
				t.Reset(500 * time.Millisecond)
			}
		}
	}(fd)
	return sess.Wait()
}

// Close close
func (c *Client) Close() error {
	return c.client.Close()
}

// GetprobePasswd 获取探测到的密码
func (c *Client) GetprobePasswd() string {
	return c.pass
}
