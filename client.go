package ssh

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Client client
type Client struct {
	opts   Options
	client *ssh.Client
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

// Connect dail connect
func (c *Client) Connect() error {
	var err error
	users := []string{c.opts.Username}
	for _, user := range users {
		var auths []ssh.AuthMethod
		if auths, err = c.authMethods(); err != nil {
			return fmt.Errorf("auth: %w", err)
		}

		config := &ssh.ClientConfig{
			User:            user,
			Auth:            auths,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         time.Duration(c.opts.Timeout) * time.Second,
		}
		addr := fmt.Sprintf("%s:%d", c.opts.IP, c.opts.Port)
		if c.client, err = ssh.Dial("tcp", addr, config); err == nil {
			return nil
		}
	}
	return err
}

// CmdOutBytes cmd out bytes
func (c *Client) CmdOutBytes(cmd string) ([]byte, error) {
	sess, err := c.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("create ssh session: %w", err)
	}
	defer sess.Close()
	for k, v := range c.opts.Env {
		if err = sess.Setenv(k, v); err != nil {
			return nil, fmt.Errorf("AcceptEnv? Setenv[%s]: %w", k, err)
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
			return nil, fmt.Errorf("request pty: %w", err)
		}
	}
	return sess.CombinedOutput(cmd)
}

// Upload upload
func (c *Client) Upload(src, dest string, mode os.FileMode) error {
	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return fmt.Errorf("建立sftp出错: %w", err)
	}
	defer sftpClient.Close()

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("读取本地文件[%s]出错: %w", src, err)
	}
	defer srcFile.Close()

	destFile, err := sftpClient.Create(dest)
	if err != nil {
		return fmt.Errorf("创建远程文件[%s]出错: %w", dest, err)
	}
	defer destFile.Close()

	size := 0
	buf := make([]byte, 1024*1024)
	for {
		n, err := srcFile.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("上传文件read出错#1: %w", err)
		}
		if n == 0 {
			break
		}
		if _, err := destFile.Write(buf[:n]); err != nil {
			return fmt.Errorf("上传文件write出错#2: %w", err)
		}
		size += n
	}
	return destFile.Chmod(mode)
}

// Close close
func (c *Client) Close() error {
	return c.client.Close()
}

// 解析登录方式
func (c *Client) authMethods() (authMethods []ssh.AuthMethod, err error) {
	passwords := []string{c.opts.Password}

	if length := len(passwords); length != 0 {
		n := 0
		authMethod := ssh.RetryableAuthMethod(ssh.PasswordCallback(func() (string, error) {
			password := passwords[n]
			n++
			return password, nil
		}), length)
		authMethods = append(authMethods, authMethod)
	}

	if c.opts.Key != "" {
		var keyBytes []byte
		keyBytes, err = base64.StdEncoding.DecodeString(strings.TrimSpace(c.opts.Key)) // private key content, must base64 code
		if err != nil {
			filepath := strings.Replace(c.opts.Key, "~", os.Getenv("HOME"), -1)
			keyBytes, err = ioutil.ReadFile(filepath) //private key file
		}
		if err != nil {
			return authMethods, err
		}
		// Create the Signer for this private key.
		var signer ssh.Signer
		if c.opts.Password == "" {
			signer, err = ssh.ParsePrivateKey(keyBytes)
		} else {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(c.opts.Password))
		}
		if err != nil {
			return authMethods, err
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}
	if c.opts.QAs != nil {
		answers := keyboardInteractive(c.opts.QAs)
		authMethods = append(authMethods, ssh.KeyboardInteractive(answers.Challenge))
	}
	return authMethods, nil
}
