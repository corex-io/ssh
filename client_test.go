package ssh_test

import (
	"testing"

	"github.com/corex-io/ssh"
)

func Test_login(t *testing.T) {
	client := ssh.New(ssh.IP("127..0.0.1"), ssh.Port(36000), ssh.Username("root"), ssh.Password("111"), ssh.Timeout(3))
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
func Test_FastConnect(t *testing.T) {
	client := ssh.New(ssh.IP("127..0.0.1"), ssh.Port(36000), ssh.Username("root"), ssh.Password("111"), ssh.Timeout(3))

	err := client.FastConnect()
	if err != nil {
		t.Logf("%v", err)
		return
	}
	t.Logf("err=%v, password=%s", err, client.GetprobePasswd())

}
