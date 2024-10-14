package ssh_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/ostools"
	"github.com/YuukanOO/seelf/pkg/ssh"
)

func Test_FileConfigurator(t *testing.T) {
	sut := func(initialConfigContent string) (ssh.Configurator, string) {
		path := filepath.Join(id.New[string](), "config")

		t.Cleanup(func() {
			os.RemoveAll(filepath.Dir(path))
		})

		if initialConfigContent != "" {
			_ = ostools.WriteFile(path, []byte(initialConfigContent))
		}

		return ssh.NewFileConfigurator(path), path
	}

	t.Run("should be able to create a new ssh config if none is found and append the host", func(t *testing.T) {
		configurator, path := sut("")

		assert.Nil(t, configurator.Upsert(ssh.Connection{
			Host: "example.com",
		}))
		assert.FileContentEquals(t, `Host example.com
StrictHostKeyChecking accept-new
`, path)
	})

	t.Run("should correctly append a host to an existing config file", func(t *testing.T) {
		configurator, path := sut("Host example.com\nUser root\n")

		assert.Nil(t, configurator.Upsert(ssh.Connection{
			Host: "somewhere.com",
			User: monad.Value("user"),
			Port: monad.Value(2222),
		}))
		assert.FileContentEquals(t, `Host example.com
User root
Host somewhere.com
StrictHostKeyChecking accept-new
User user
Port 2222
`, path)
	})

	t.Run("should correctly update an existing host", func(t *testing.T) {
		configurator, path := sut(`Host example.com
User root
Host somewhere.com
StrictHostKeyChecking accept-new
User user
Port 2222
`)

		assert.Nil(t, configurator.Upsert(ssh.Connection{
			Host: "somewhere.com",
			User: monad.Value("root"),
			Port: monad.Value(22),
		}))
		assert.FileContentEquals(t, `Host example.com
User root
Host somewhere.com
StrictHostKeyChecking accept-new
User root
Port 22
`, path)
	})

	t.Run("should update an host only if the identifier match", func(t *testing.T) {
		configurator, path := sut(`Host example.com
User root
Host example.com #my-identifier
User john
`)

		assert.Nil(t, configurator.Upsert(ssh.Connection{
			Identifier: "my-identifier",
			Host:       "another.com",
			User:       monad.Value("john"),
			Port:       monad.Value(2222),
		}))
		assert.FileContentEquals(t, `Host example.com
User root
Host another.com #my-identifier
StrictHostKeyChecking accept-new
User john
Port 2222
`, path)
	})

	t.Run("should write the private key if set", func(t *testing.T) {
		configurator, path := sut("")
		expectedKeyPath := filepath.Join(filepath.Dir(path), "privkeyfilename")

		assert.Nil(t, configurator.Upsert(ssh.Connection{
			Host: "example.com",
			PrivateKey: monad.Value(ssh.ConnectionKey{
				Name: "privkeyfilename",
				Key:  "privkeycontent",
			}),
		}))
		assert.FileContentEquals(t, fmt.Sprintf(`Host example.com
StrictHostKeyChecking accept-new
IdentityFile %s
IdentitiesOnly yes
`, expectedKeyPath), path)
		assert.FileContentEquals(t, "privkeycontent", expectedKeyPath)
	})

	t.Run("should remove the old private key if it was set", func(t *testing.T) {
		configurator, path := sut("")
		oldKeyPath := filepath.Join(filepath.Dir(path), "oldkeyfilename")
		newKeyPath := filepath.Join(filepath.Dir(path), "newkeyfilename")
		assert.Nil(t, configurator.Upsert(ssh.Connection{
			Host: "example.com",
			PrivateKey: monad.Value(ssh.ConnectionKey{
				Name: "oldkeyfilename",
				Key:  "oldprivkeycontent",
			}),
		}))

		assert.Nil(t, configurator.Upsert(ssh.Connection{
			Host: "example.com",
			PrivateKey: monad.Value(ssh.ConnectionKey{
				Name: "newkeyfilename",
				Key:  "newprivkeycontent",
			}),
		}))
		assert.FileContentEquals(t, fmt.Sprintf(`Host example.com
StrictHostKeyChecking accept-new
IdentityFile %s
IdentitiesOnly yes
`, newKeyPath), path)
		assert.FileContentEquals(t, "newprivkeycontent", newKeyPath)
		assert.FileContentEquals(t, "", oldKeyPath)
	})

	t.Run("should do nothing if trying to delete an host and no config file exist", func(t *testing.T) {
		configurator, _ := sut("")

		assert.Nil(t, configurator.Remove("test"))
	})

	t.Run("should correctly remove an host", func(t *testing.T) {
		configurator, path := sut(`Host example.com
User root
Host example.com #my-identifier
User john
`)

		assert.Nil(t, configurator.Remove(""))
		assert.FileContentEquals(t, `Host example.com #my-identifier
User john
`, path)
	})

	t.Run("should correctly remove an host with a specific identifier", func(t *testing.T) {
		configurator, path := sut(`Host example.com
User root
Host example.com #my-identifier
User john
`)

		assert.Nil(t, configurator.Remove("my-identifier"))
		assert.FileContentEquals(t, `Host example.com
User root
`, path)
	})

	t.Run("should remove the private key attached to the host being removed", func(t *testing.T) {
		configurator, path := sut("")
		keyPath := filepath.Join(filepath.Dir(path), "privkeyfilename")
		assert.Nil(t, configurator.Upsert(ssh.Connection{
			Host: "example.com",
			PrivateKey: monad.Value(ssh.ConnectionKey{
				Name: "privkeyfilename",
				Key:  "privkeycontent",
			}),
		}))

		assert.Nil(t, configurator.Remove(""))
		assert.FileContentEquals(t, "", path)
		assert.FileContentEquals(t, "", keyPath)
	})
}
