package ssh

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/ostools"
	"github.com/kevinburke/ssh_config"
)

type (
	// Represents a configurator used to manipulate an ssh config file and wrap
	// common stuff to make working with ssh easier.
	Configurator interface {
		Upsert(conn Connection) error   // Ensure the given connection is present in the config and write private key if given.
		Remove(identifier string) error // Remove an entry identified with the given value. It will also remove the private key referenced if found.
	}

	fileConfigurator struct {
		mu   sync.Mutex
		dir  string
		path string
	}

	Connection struct {
		Identifier string // Custom identifier used to identify a specific connection, if set, it will take precedence over the host
		Host       Host
		User       monad.Maybe[string]
		Port       monad.Maybe[int]
		PrivateKey monad.Maybe[ConnectionKey]
	}

	ConnectionKey struct {
		Name string
		Key  PrivateKey
	}
)

// Instantiate a new configurator which will manipulate the given ssh config file.
func NewFileConfigurator(path string) Configurator {
	return &fileConfigurator{
		dir:  filepath.Dir(path),
		path: path,
	}
}

// Configure an ssh config by editing the config file and writing
// private key to the appropriate file.
func (c *fileConfigurator) Upsert(conn Connection) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	f, err := os.Open(c.path)

	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var sshConfig *ssh_config.Config

	// No config file found, create a new config
	if f == nil {
		sshConfig = &ssh_config.Config{}
	} else {
		if sshConfig, err = ssh_config.Decode(f); err != nil {
			return err
		}
	}

	var (
		sshHost        *ssh_config.Host
		hostname       = conn.Host.String()
		hasIdentifier  = conn.Identifier != ""
		oldPrivKeyPath string // old private key path if the node exists
	)

	// Try to retrieve an already existing host
	for _, host := range sshConfig.Hosts {
		if host.IsImplicit() ||
			(hasIdentifier && host.EOLComment != conn.Identifier) ||
			(!hasIdentifier && !host.Matches(hostname)) {
			continue
		}

		sshHost = host

		// Check if it had a private key set
		for _, node := range sshHost.Nodes {
			if kv, isKV := node.(*ssh_config.KV); isKV && kv.Key == "IdentityFile" {
				oldPrivKeyPath = kv.Value
				break
			}
		}

		break
	}

	// Creates it if its nil
	if sshHost == nil {
		sshHost = &ssh_config.Host{}
		sshConfig.Hosts = append(sshConfig.Hosts, sshHost)
	}

	// Update the host entry
	pattern, err := ssh_config.NewPattern(hostname)

	if err != nil {
		return err
	}

	sshHost.Patterns = []*ssh_config.Pattern{pattern}
	sshHost.EOLComment = conn.Identifier
	sshHost.Nodes = make([]ssh_config.Node, 0, 5)
	sshHost.Nodes = append(sshHost.Nodes, &ssh_config.KV{
		Key:   "StrictHostKeyChecking",
		Value: "accept-new", // We still want to prevent MiTM attacks!
	})

	if user, isSet := conn.User.TryGet(); isSet {
		sshHost.Nodes = append(sshHost.Nodes, &ssh_config.KV{
			Key:   "User",
			Value: user,
		})
	}

	if port, isSet := conn.Port.TryGet(); isSet {
		sshHost.Nodes = append(sshHost.Nodes, &ssh_config.KV{
			Key:   "Port",
			Value: strconv.Itoa(port),
		})
	}

	// Remove the old private key if it was set
	if err = os.RemoveAll(oldPrivKeyPath); err != nil {
		return err
	}

	if privKey, hasPrivKey := conn.PrivateKey.TryGet(); hasPrivKey {
		privateKeyPath := filepath.Join(c.dir, privKey.Name)

		if err := ostools.WriteFile(privateKeyPath, []byte(privKey.Key), 0600); err != nil {
			return err
		}

		sshHost.Nodes = append(sshHost.Nodes,
			&ssh_config.KV{
				Key:   "IdentityFile",
				Value: privateKeyPath,
			}, &ssh_config.KV{
				Key:   "IdentitiesOnly",
				Value: "yes",
			})
	}

	return ostools.WriteFile(c.path, []byte(sshConfig.String()), 0644)
}

func (c *fileConfigurator) Remove(identifier string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	f, err := os.Open(c.path)

	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// No ssh config file, nothing to do
	if f == nil {
		return nil
	}

	sshConfig, err := ssh_config.Decode(f)

	if err != nil {
		return err
	}

	// Remove the line matching the given identifier
	for i, host := range sshConfig.Hosts {
		if host.IsImplicit() || host.EOLComment != identifier {
			continue
		}

		// Remove the private key from the file system if any
		for _, node := range host.Nodes {
			if kv, isKV := node.(*ssh_config.KV); isKV && kv.Key == "IdentityFile" {
				if err = os.RemoveAll(kv.Value); err != nil {
					return err
				}
			}
		}

		// Remove the host from sshConfig.Hosts
		sshConfig.Hosts = append(sshConfig.Hosts[:i], sshConfig.Hosts[i+1:]...)
		break
	}

	return ostools.WriteFile(c.path, []byte(sshConfig.String()), 0644)
}
