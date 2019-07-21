package test

import (
	"bytes"
	"os/exec"
	"os/user"
	"strings"
	"syscall"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestSSHExecServer(t *testing.T) {

	s := NewTestSSHExecServer(t, func(req *ssh.Request, channel ssh.Channel) {

		cmd := exec.Command("sh", "-c", string(req.Payload))
		cmd.Stdin = channel
		cmd.Stdout = channel
		cmd.Stderr = channel

		if err := cmd.Run(); err != nil {
			if msg, ok := err.(*exec.ExitError); ok {
				code := byte(msg.Sys().(syscall.WaitStatus).ExitStatus())
				channel.SendRequest("exit-status", false, []byte{0, 0, 0, code})
			} else {
				channel.SendRequest("exit-status", false, []byte{0, 0, 0, 1})
			}
			t.Error(err)
		} else {
			channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
		}

		channel.Close()

	})
	defer s.Close()

	usr, _ := user.Current()
	client, err := ssh.Dial("tcp", s.Host, &ssh.ClientConfig{
		User:            usr.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		t.Error(err)
	}
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		t.Error(err)
	}
	defer sess.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sess.Stdout = stdout
	sess.Stderr = stderr

	if err := sess.Run("whoami"); err != nil {
		t.Error(err)
	}

	if username := strings.Trim(stdout.String(), "\t\r\n "); username != usr.Username {
		t.Errorf("Expected %q, got %q for output of `whoami`", usr.Username, username)
	}

	if stderr.Len() > 0 {
		t.Errorf("Expected 0 bytes from stderr, got %q", stderr.String())
	}

}
