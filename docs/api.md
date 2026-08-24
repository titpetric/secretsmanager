# Package secretsmanager

```go
import (
	"github.com/titpetric/secretsmanager"
}
```
Package secretsmanager stores secrets in a repository, encrypted at rest
with a key kept out of it.

Secrets live in one workspace, which is a directory holding a
.secrets.json, and are addressed by the environment variable name they
produce, so DB_DSN, db_dsn and db-dsn all name one secret:

```go
secrets, err := secretsmanager.NewStorage(secretsmanager.NewOptionsFromEnv())
if err != nil {
	return err
}

secret, err := secrets.Get(ctx, "DB_DSN")
if err != nil {
	return err
}
fmt.Println(secret.Value)
```

NewOptionsFromEnv reads SECRETSMANAGER_WORKSPACE and the 32 character
SECRETSMANAGER_KEY. An Options built by hand takes the key directly, for
a caller which doesn't keep it in the environment.

Reading never writes the file: only Set does, and it rewrites one secret,
leaving the ciphertext of the others as it found it. Nothing is held in
package state, so two storages can serve two workspaces at once, each
under its own key.

## Types

```go
// Options configure a storage. The zero value reads the current directory,
// with the key from the environment.
type Options struct {
	// Workspace is the directory holding the secrets file. An empty
	// Workspace is the current directory.
	Workspace	string

	// Key is the 32 byte key the values are encrypted with. A nil Key is
	// read from SECRETSMANAGER_KEY when the secrets are first used.
	Key	[]byte
}
```

```go
// Secret is one stored secret: the name it's used under, its value, and an
// ID for it.
type Secret struct {
	// ID is a ULID for secrets this tool created. It's kept as a string so
	// the UUIDs written by earlier versions still load.
	ID	string
	Name	string
	Value	string

	// raw holds the ciphertext this value was last read or written as, and
	// is empty for a value which hasn't been stored yet. Writing it back
	// verbatim keeps untouched secrets out of the diff, as every encrypt
	// produces a new IV and with it a different ciphertext.
	raw	string
}
```

```go
// Storage holds the secrets of one workspace.
//
// Every call takes a context, as a driver other than the local file may talk
// to a remote host to serve it. Reading and writing the secrets is the
// driver's business: Get and List fetch what they need, Set has stored the
// secret by the time it returns.
type Storage interface {
	// String describes where the secrets are kept, for error messages.
	fmt.Stringer

	// List returns the secrets in the order they're stored in.
	List(ctx context.Context) ([]*Secret, error)

	// Get returns the secret stored under a name, or ErrNotFound.
	Get(ctx context.Context, name string) (*Secret, error)

	// Set adds a secret, or replaces the value of an existing one.
	Set(ctx context.Context, name, value string) (*Secret, error)
}
```

## Vars

```go
// ErrNotFound is returned by Storage.Get for a name which isn't stored.
var ErrNotFound = errors.New("no such secret")
```

## Function symbols

- `func NewOptionsFromEnv () Options`
- `func NewStorage (options Options) (Storage, error)`

### NewOptionsFromEnv

NewOptionsFromEnv returns the options the environment configures, from
SECRETSMANAGER_WORKSPACE and SECRETSMANAGER_KEY.

A key which isn't set, or is the wrong length, is left for the storage to
report when it reads the secrets. It isn't an error to be missing one
yet: init generates the key, and has to run before there is one.

```go
func NewOptionsFromEnv () Options
```

### NewStorage

NewStorage returns the storage driver for a workspace, which is a
directory today. One carrying a URL scheme names a driver which doesn't
exist yet, and saying so beats trying to open it as a path.

```go
func NewStorage (options Options) (Storage, error)
```


