package shaw

import (
	"io/ioutil"

	"github.com/emptyinterface/knownhosts"
	"golang.org/x/crypto/ssh"
)

func NewSSHClientConfig(user, keypath string) (*ssh.ClientConfig, error) {

	data, err := ioutil.ReadFile(keypath)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return nil, err
	}

	c := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
	}

	return c, nil

}

func EnsureKnownHosts(config *ssh.ClientConfig, knownHosts string) {
	key := knownhosts.NewHostKeyFile(knownHosts)
	check := knownhosts.NewHostKeyChecker(key)
	config.HostKeyCallback = check.Check
}
