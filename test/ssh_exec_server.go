package test

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"

	"golang.org/x/crypto/ssh"
)

type (
	SSHExecServer struct {
		Host string
		l    net.Listener
	}
)

var (
	signer ssh.Signer

	sshToOS = map[ssh.Signal]os.Signal{
		ssh.SIGABRT: syscall.SIGABRT,
		ssh.SIGALRM: syscall.SIGALRM,
		ssh.SIGFPE:  syscall.SIGFPE,
		ssh.SIGHUP:  syscall.SIGHUP,
		ssh.SIGILL:  syscall.SIGILL,
		ssh.SIGINT:  syscall.SIGINT,
		ssh.SIGKILL: syscall.SIGKILL,
		ssh.SIGPIPE: syscall.SIGPIPE,
		ssh.SIGQUIT: syscall.SIGQUIT,
		ssh.SIGSEGV: syscall.SIGSEGV,
		ssh.SIGTERM: syscall.SIGTERM,
		ssh.SIGUSR1: syscall.SIGUSR1,
		ssh.SIGUSR2: syscall.SIGUSR2,
	}
)

func init() {
	const privateBytes = `-----BEGIN RSA PRIVATE KEY-----
MIIBzAIBAAJhAOIAwMVZCOtUEjtrGsv0CkDTYgGGeS4z5sgtaTrwg/6gWYMtTSWc
zgQ9wmpdo2rNZypUUXy2cXzAyiaUwp4jXSctPYVYErLk0KGycK6SaJogu7HAemiZ
3TLn8QkfODbakQIDAQABAmEA4PDY7VNx0jAKOYOf1zGdZuo9mMEMKdVUtRalrxkm
dy+ICEz1hSMt1gDWWWG7vhiS4ALlW/TKFMP6E4rkiqG+tQ3thrdEwyeFFQBzBoyq
dhb7Dgipez5ELh3282g8dWsxAjEA98oYjJ4Gds7gCFenc8daNxdhSdKu3GVY32kV
aV8/Quhpq2lTywYlsvRs7bN6u3WtAjEA6X3ZuxGt55h2AHhwO9mzU9DS3KPP15iA
i0zieVb/Tg3i/iykHy5kkRzzuujQm6z1AjEA5NT7ROkvGQtF9A5W82I4G0Z5Lz7l
A16I65FVF8HBX13ZMFaN7qGXsSNvcTld777lAjEA5O1/jOrIl0nkaJGteQD50jPs
imgSYFAluG6pnk6uAtmatZsPT4MtFxpL3fZmkjwBAjAZA2joUKAAW//N3zHzlZNO
6CaM8izhmFh3Bn1KM1ByPzpoHcvIzScvS9f4j9iVOMw=
-----END RSA PRIVATE KEY-----`
	var err error
	signer, err = ssh.ParsePrivateKey([]byte(privateBytes))
	if err != nil {
		log.Fatal(err)
	}
}

func NewTestSSHExecServer(t *testing.T, f func(reqs <-chan *ssh.Request, channel ssh.Channel) error) *SSHExecServer {

	l, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Error(err)
		t.Fatal("failed to acquire tcp listener")
	}

	go func() {

		for {
			conn, err := l.Accept()
			if err != nil {
				// closing the listener throws an error, we can safely ignore it
				if strings.HasSuffix(err.Error(), "use of closed network connection") {
					return
				}
				t.Error(err)
				t.Fatal("failed to accept incoming connection")
			}
			go func(conn net.Conn) {
				defer conn.Close()

				// Before use, a handshake must be performed on the incoming
				config := &ssh.ServerConfig{NoClientAuth: true}
				config.AddHostKey(signer)
				_, chans, reqs, err := ssh.NewServerConn(conn, config)
				if err != nil {
					t.Error(err)
					t.Fatal("failed to handshake")
				}

				// The incoming Request channel must be serviced.
				go ssh.DiscardRequests(reqs)

				for c := range chans {

					if t := c.ChannelType(); t != "session" {
						c.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
						continue
					}

					channel, requests, err := c.Accept()
					if err != nil {
						t.Error(err)
						t.Fatal("could not accept channel.")
					}

					if err := f(requests, channel); err != nil {
						t.Error(err)
						return
					}

					channel.Close()

				}

			}(conn)
		}

	}()

	return &SSHExecServer{
		Host: l.Addr().String(),
		l:    l,
	}

}

func (s *SSHExecServer) Close() error {
	return s.l.Close()
}

// ExecFunc handles "exec", "env", and "signal" requests
func ExecFunc(reqs <-chan *ssh.Request, channel ssh.Channel) error {

	var (
		env []string
		cmd *exec.Cmd
	)

	// pretty.Println(req)

	for req := range reqs {

		switch req.Type {
		case "env":
			// RFC 4254 Section 6.4.
			var setenvRequest struct {
				Name  string
				Value string
			}
			if err := ssh.Unmarshal(req.Payload, &setenvRequest); err != nil {
				req.Reply(false, nil)
				return err
			}
			env = append(env, fmt.Sprintf("%s=%s", setenvRequest.Name, setenvRequest.Value))
			req.Reply(true, nil)
		case "signal":
			// RFC 4254 Section 6.9.
			var signalMsg struct {
				Signal string
			}
			if err := ssh.Unmarshal(req.Payload, &signalMsg); err != nil {
				req.Reply(false, nil)
				return err
			}
			req.Reply(true, nil)
			if cmd != nil && cmd.Process != nil {
				sig := sshToOS[ssh.Signal(signalMsg.Signal)]
				if err := cmd.Process.Signal(sig); err != nil {
					return err
				}
			}
		case "exec":

			// RFC 4254 Section 6.5.
			var execMsg struct {
				Command string
			}

			if err := ssh.Unmarshal(req.Payload, &execMsg); err != nil {
				req.Reply(false, nil)
				return err
			}
			req.Reply(true, nil)

			cmd = exec.Command("sh", "-c", string(execMsg.Command))
			cmd.Env = env
			cmd.Stdin = channel
			cmd.Stdout = channel
			cmd.Stderr = channel

			if err := cmd.Start(); err != nil {
				return err
			}

			go func() {
				if err := cmd.Wait(); err != nil {
					if msg, ok := err.(*exec.ExitError); ok {
						code := byte(msg.Sys().(syscall.WaitStatus).ExitStatus())
						channel.SendRequest("exit-status", false, []byte{0, 0, 0, code})
					} else {
						channel.SendRequest("exit-status", false, []byte{0, 0, 0, 1})
					}

				}
				channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
				channel.Close()
			}()
		}

	}

	return nil

}
