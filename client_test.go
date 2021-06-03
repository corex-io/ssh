package ssh_test

import (
	"testing"

	"github.com/corex-io/ssh"
)

func Test_login(t *testing.T) {
	client := ssh.New(ssh.IP("127.0.0.1"), ssh.Port(22), ssh.Username("root"), ssh.Password("root"))
	if err := client.Connect(); err != nil {
		t.Error(err)
		return
	}
	out, err := client.CmdOutBytes("ifconfig")
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(string(out))
}
