package shaw

import (
	"bufio"
	"bytes"
	"os/user"
	"testing"

	"github.com/emptyinterface/extio"
	"github.com/emptyinterface/shaw/test"
	"golang.org/x/crypto/ssh"
)

func TestPoolExec(t *testing.T) {

	const poolsize = 10

	var servers []*test.SSHExecServer

	for i := 0; i < poolsize; i++ {
		server := test.NewTestSSHExecServer(t, test.ExecFunc)
		defer server.Close()
		servers = append(servers, server)
	}

	u, _ := user.Current()
	pool := NewSSHPool()

	for i := 0; i < poolsize; i++ {
		config := &ssh.ClientConfig{
			User:            u.Username,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
		pool.AddClient(NewSSHClient(servers[i].Host, config))
	}

	var stdout []string

	for _, report := range pool.NewCommandSession(&Command{
		Executor: BinBashExecutor,
		Command:  "whoami",
		Stdout: extio.NewScannerWriter(bufio.ScanLines, 1<<10, func(token []byte) error {
			stdout = append(stdout, string(token))
			return nil
		}),
		Stderr: &bytes.Buffer{},
	}).Run() {
		if report.err != nil {
			t.Error(report.err)
		}
		if report.start.IsZero() {
			t.Errorf("Expected non-zero start time")
		}
		if report.end.IsZero() {
			t.Errorf("Expected non-zero end time")
		}
		if stderr := report.cmd.Stderr.(*bytes.Buffer); stderr.Len() > 0 {
			t.Errorf("Expected 0 bytes from stderr, got %q", stderr.String())
		}
	}

	if len(stdout) != poolsize {
		t.Errorf("Expected %d lines back, got %d", poolsize, len(stdout))
	}

	for _, line := range stdout {
		if line != u.Username {
			t.Errorf("Expected %q, got %q", u.Username, line)
		}
	}

}
