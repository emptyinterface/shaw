package shaw

import (
	"time"

	"github.com/emptyinterface/extio"
	"golang.org/x/crypto/ssh"
)

type (
	SSHPool struct {
		clients []*SSHClient
	}

	PoolCommandSession struct {
		cmd        *Command
		sessions   []*CommandSession
		reports    []*CommandReport
		start, end time.Time
	}

	CommandReport struct {
		session    *CommandSession
		cmd        *Command
		err        error
		start, end time.Time
	}
)

func NewSSHPool(clients ...*SSHClient) *SSHPool {
	return &SSHPool{
		clients: clients,
	}
}

func (p *SSHPool) AddClient(client *SSHClient) {
	p.clients = append(p.clients, client)
}

func (p *SSHPool) NewCommandSession(cmd *Command) *PoolCommandSession {

	ps := &PoolCommandSession{
		cmd: cmd,
	}

	var bc *extio.Broadcaster
	if cmd.Stdin != nil {
		bc = extio.NewBroadcaster(cmd.Stdin)
	}

	for _, client := range p.clients {
		// make copy
		// all vars but stdin can be copied
		c := *cmd
		if bc != nil {
			c.Stdin = bc.NewReader()
		}
		ps.sessions = append(ps.sessions, client.NewCommandSession(&c))
	}

	return ps

}

func (ps *PoolCommandSession) Start() {
	ps.start = time.Now()
	for _, s := range ps.sessions {
		ps.reports = append(ps.reports, &CommandReport{
			session: s,
			cmd:     s.cmd,
			start:   time.Now(),
			err:     s.Start(),
		})
	}
}

func (ps *PoolCommandSession) Wait() []*CommandReport {
	for i, s := range ps.sessions {
		if err := s.Wait(); err != nil && ps.reports[i].err == nil {
			ps.reports[i].err = err
		}
		ps.reports[i].end = s.end
	}
	ps.end = time.Now()
	return ps.reports
}

func (ps *PoolCommandSession) Run() []*CommandReport {
	ps.Start()
	return ps.Wait()
}

func (ps *PoolCommandSession) Signal(sig ssh.Signal) []*CommandReport {
	for i, s := range ps.sessions {
		if err := s.Signal(sig); err != nil && ps.reports[i].err == nil {
			ps.reports[i].err = err
		}
	}
	return ps.Wait()
}
