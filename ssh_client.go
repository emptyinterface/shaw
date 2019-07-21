package shaw

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type (
	SSHClient struct {
		Host        string
		DialTimeout time.Duration

		config *ssh.ClientConfig
	}

	Command struct {
		Env      []EnvVariable
		Executor string
		Command  string

		Stdin  io.Reader
		Stdout io.Writer
		Stderr io.Writer
	}

	EnvVariable struct {
		Name, Value string
	}

	CommandSession struct {
		client *SSHClient
		cmd    *Command
		sess   *ssh.Session
		ready  sync.WaitGroup

		start, end time.Time
	}
)

const (
	DefaultDialTimeout = 2 * time.Second

	BinBashExecutor          = `/bin/bash -c %q`
	BinBashStdinExecutor     = `/bin/bash -s`
	SudoBinBashExecutor      = `/usr/bin/sudo /bin/bash -c %q`
	SudoBinBashStdinExecutor = `/usr/bin/sudo /bin/bash -s`
)

var (
	ErrSessionNotStarted = errors.New("ssh session not started yet")
)

func NewBashCommand(cmdf string, args ...interface{}) *Command {
	return &Command{
		Executor: BinBashExecutor,
		Command:  fmt.Sprintf(cmdf, args...),
	}
}

func (c *Command) StdinPipe() io.WriteCloser {
	r, w := io.Pipe()
	c.Stdin = r
	return w
}

func (c *Command) StdoutPipe() io.ReadCloser {
	r, w := io.Pipe()
	c.Stdout = w
	return r
}

func (c *Command) StderrPipe() io.ReadCloser {
	r, w := io.Pipe()
	c.Stderr = w
	return r
}

func NewSSHClient(host string, config *ssh.ClientConfig) *SSHClient {
	return &SSHClient{
		Host:        host,
		DialTimeout: DefaultDialTimeout,
		config:      config,
	}
}

func (c *SSHClient) NewCommandSession(cmd *Command) *CommandSession {
	s := &CommandSession{
		client: c,
		cmd:    cmd,
	}
	s.ready.Add(1)
	return s
}

func (cs *CommandSession) Start() error {

	cs.start = time.Now()
	defer func() { cs.end = time.Now() }()

	conn, err := net.DialTimeout("tcp", cs.client.Host, cs.client.DialTimeout)
	if err != nil {
		return err
	}

	c, chans, reqs, err := ssh.NewClientConn(conn, cs.client.Host, cs.client.config)
	if err != nil {
		return err
	}

	cs.sess, err = ssh.NewClient(c, chans, reqs).NewSession()
	if err != nil {
		return err
	}
	cs.ready.Done()

	for _, v := range cs.cmd.Env {
		if err := cs.sess.Setenv(v.Name, v.Value); err != nil {
			return err
		}
	}

	cs.sess.Stdin = cs.cmd.Stdin
	cs.sess.Stdout = cs.cmd.Stdout
	cs.sess.Stderr = cs.cmd.Stderr

	var cmd string
	if len(cs.cmd.Command) > 0 {
		cmd = fmt.Sprintf(cs.cmd.Executor, cs.cmd.Command)
	} else {
		cmd = cs.cmd.Executor
	}

	cs.start = time.Now()

	return cs.sess.Start(cmd)

}

func (cs *CommandSession) Wait() error {
	defer func() { cs.end = time.Now() }()
	cs.ready.Wait()
	return cs.sess.Wait()
}

func (cs *CommandSession) Run() error {
	if err := cs.Start(); err != nil {
		return err
	}
	return cs.Wait()
}

func (cs *CommandSession) Signal(sig ssh.Signal) error {
	cs.ready.Wait()
	return cs.sess.Signal(sig)
}

func (cs *CommandSession) Close() error {
	cs.ready.Wait()
	return cs.sess.Close()
}
