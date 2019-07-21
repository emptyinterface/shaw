package shaw

import (
	"bufio"
	"bytes"
	"fmt"
	"os/user"
	"strings"
	"testing"
	"time"

	"github.com/emptyinterface/shaw/test"
	"golang.org/x/crypto/ssh"
)

func TestSSHClientExec(t *testing.T) {

	server := test.NewTestSSHExecServer(t, test.ExecFunc)
	defer server.Close()

	u, _ := user.Current()
	config := &ssh.ClientConfig{
		User:            u.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client := NewSSHClient(server.Host, config)

	var stdout, stderr bytes.Buffer

	if err := client.NewCommandSession(&Command{
		Executor: BinBashExecutor,
		Command:  "whoami",
		Stdout:   &stdout,
		Stderr:   &stderr,
	}).Run(); err != nil {
		t.Error(err)
	}

	if username := strings.TrimSpace(stdout.String()); username != u.Username {
		t.Errorf("Expected %q, got %q for output of `whoami`", u.Username, username)
	}

	if stderr.Len() > 0 {
		t.Errorf("Expected 0 bytes from stderr, got %q", stderr.String())
	}

}

func TestSSHClientExecStdin(t *testing.T) {

	server := test.NewTestSSHExecServer(t, test.ExecFunc)
	defer server.Close()

	u, _ := user.Current()
	config := &ssh.ClientConfig{
		User:            u.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client := NewSSHClient(server.Host, config)

	var stdout, stderr bytes.Buffer

	if err := client.NewCommandSession(&Command{
		Executor: BinBashStdinExecutor,
		Stdin:    bytes.NewBufferString("whoami"),
		Stdout:   &stdout,
		Stderr:   &stderr,
	}).Run(); err != nil {
		t.Error(err)
	}

	if username := strings.TrimSpace(stdout.String()); username != u.Username {
		t.Errorf("Expected %q, got %q for output of `whoami`", u.Username, username)
	}

	if stderr.Len() > 0 {
		t.Errorf("Expected 0 bytes from stderr, got %q", stderr.String())
	}

}

func TestSSHClientSetEnv(t *testing.T) {

	var envVars = []EnvVariable{
		{Name: "KEY1", Value: "1111"},
		{Name: "KEY2", Value: "2222"},
		{Name: "SPACE_KEY", Value: "AAA AAA"},
		{Name: "COMMAND_KEY", Value: "ls -alh"},
		{Name: "CONCAT_KEY", Value: "$KEY1:$KEY2"},
	}

	server := test.NewTestSSHExecServer(t, test.ExecFunc)
	defer server.Close()

	config := &ssh.ClientConfig{
		User:            "test",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client := NewSSHClient(server.Host, config)

	var stdout, stderr bytes.Buffer

	if err := client.NewCommandSession(&Command{
		Executor: BinBashExecutor,
		Command:  "env",
		Env:      envVars,
		Stdout:   &stdout,
		Stderr:   &stderr,
	}).Run(); err != nil {
		t.Error(err)
	}

	env := map[string]bool{}
	sc := bufio.NewScanner(&stdout)

	for sc.Scan() {
		env[sc.Text()] = true
	}
	if sc.Err() != nil {
		t.Error(sc.Err())
	}

	for _, kv := range envVars {
		v := fmt.Sprintf("%s=%s", kv.Name, kv.Value)
		if !env[v] {
			t.Errorf("Expected %q, but none found in `env` output", v)
		}
	}

	if stderr.Len() > 0 {
		t.Errorf("Expected 0 bytes from stderr, got %q", stderr.String())
	}

}

func TestSSHClientSignal(t *testing.T) {

	server := test.NewTestSSHExecServer(t, test.ExecFunc)
	defer server.Close()

	u, _ := user.Current()
	config := &ssh.ClientConfig{
		User:            u.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client := NewSSHClient(server.Host, config)

	sess := client.NewCommandSession(NewBashCommand("sleep 3"))

	if err := sess.Start(); err != nil {
		t.Error(err)
	}

	time.Sleep(200 * time.Millisecond)

	if err := sess.Signal(ssh.SIGTERM); err != nil {
		t.Error(err)
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("Expected sleep to exit by now.  SIGTERM seems to have failed.")
		}
	}()

	if err := sess.Wait(); err != nil {
		t.Error(err)
	}
	close(done)

}
