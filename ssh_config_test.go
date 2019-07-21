package shaw

import (
	"bytes"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestNewSSHClientConfig(t *testing.T) {

	var (
		user         = "user"
		privateBytes = []byte(`-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`)
	)

	signer, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		t.Error(err)
	}

	tmp, err := ioutil.TempFile("", "temp_key")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(privateBytes); err != nil {
		t.Error(err)
	}
	if err := tmp.Close(); err != nil {
		t.Error(err)
	}

	config, err := NewSSHClientConfig(user, tmp.Name())
	if err != nil {
		t.Error(err)
	}

	if config.User != user {
		t.Errorf("Expected %q, got %q", user, config.User)
	}

	if len(config.Auth) != 1 {
		t.Errorf("Expected 1 AuthMethod, got %d", len(config.Auth))
	}

	// not a great way, but use reflect to dig out the signer and compare bytes
	vals := reflect.ValueOf(config.Auth[0]).Call(nil)

	if len(vals) != 2 {
		t.Error("Expected ([]ssh.Signer, error) signature")
	}

	signers, ok := vals[0].Interface().([]ssh.Signer)
	if !ok {
		t.Errorf("Expected type []ssh.Signer, got %T", vals[0].Interface())
	}

	if len(signers) != 1 {
		t.Errorf("Expected 1 signer, got %d", len(signers))
	}

	if !bytes.Equal(signers[0].PublicKey().Marshal(), signer.PublicKey().Marshal()) {
		t.Error("public key mismatch..  woah")
	}

}
