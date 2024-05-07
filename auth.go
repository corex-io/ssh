package ssh

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

func (c *Client) keyboardAuthMethod() ssh.AuthMethod {
	answers := keyboardInteractive(c.opts.QAs)
	return ssh.KeyboardInteractive(answers.Challenge)
}

func (c *Client) keyAuthMethod() (ssh.AuthMethod, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(c.opts.Key)) // private key content, must base64 code
	if err != nil {
		filepath := strings.Replace(c.opts.Key, "~", os.Getenv("HOME"), -1)
		keyBytes, err = ioutil.ReadFile(filepath) //private key file
	}
	if err != nil {
		return nil, err
	}
	// Create the Signer for this private key.
	var signer ssh.Signer
	if c.opts.Password == "" {
		signer, err = ssh.ParsePrivateKey(keyBytes)
	} else {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(c.opts.Password))
	}
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(signer), nil
}

// 解析登录方式
func (c *Client) authMethods() ([]ssh.AuthMethod, error) {

	var auths []ssh.AuthMethod

	if c.opts.Key != "" {
		auth, err := c.keyAuthMethod()
		if err != nil {
			return nil, err
		}
		auths = append(auths, auth)
	}

	/* 密码 */

	var passwords []string
	if c.opts.Password != "" {
		passwords = append(passwords, c.opts.Password)
	}
	passwords = append(passwords, c.opts.Passwords...)

	if length := len(passwords); length != 0 {
		n := 0
		auth := ssh.RetryableAuthMethod(ssh.PasswordCallback(func() (string, error) {
			password := passwords[n]
			n++
			fmt.Println(n, password)

			return password, nil
		}), length)
		auths = append(auths, auth)
	}

	for _, password := range passwords {
		auths = append(auths, PasswordKeyboardInteractive(password))
	}

	if len(c.opts.QAs) != 0 {
		auths = append(auths, c.keyboardAuthMethod())
	}

	return auths, nil
}
