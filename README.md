# secretsmanager

I created secretsmanager to store some secrets within a repository. The
secrets are encrypted at rest, with readable keys and editable JSON, so
you can rename a key or delete it by hand. The cli tool handles the bare
minumum of requirements, and the storage behind it is importable from
`github.com/titpetric/secretsmanager` for reading the same secrets from
a Go program.

- `secretsmanager init` - generate the encryption key and name the workspace,
- `secretsmanager create` - create a new secret,
- `secretsmanager get` - print one secret value,
- `secretsmanager env` - print secrets for .env

The tool will modify `.secrets.json` as needed. Only `create` writes to
it; it creates the file if it doesn't exist, and adds or updates a single
key, leaving every other key byte for byte as it was. Creating a secret
with a name that already exists replaces its value and keeps its ID. The
`get`, `env` and `init` commands never write.

By default `.secrets.json` is read from the current directory. Set
`SECRETSMANAGER_WORKSPACE=/workspace` to use `/workspace/.secrets.json`
instead, so the tool can be run from anywhere. A workspace is a directory;
the storage behind it is a driver, and one addressed by URL is refused for
now, as the only driver is the local file.

`init` prints both variables, ready to be added to the environment:

~~~
# ./secretsmanager init
# Add the following to /etc/environment and store securely in case you need to restore
# WARN: Please, don't add/commit this key to git, as it allows decrypting all secrets.
SECRETSMANAGER_KEY=cbrt8n6fxWxRtTyaEcN7wubCR9nK7T6Z
# The directory holding .secrets.json, so the tool works from anywhere.
SECRETSMANAGER_WORKSPACE=/home/you/src/yourproject
~~~

The workspace it names is the one already configured, or the directory you
ran it in. The encryption key generated with `init` should not be commited into git.
It should be added to the ambient environment on your system, or your
deployment / CI pipeline. If you want to store it into 1password it also
wouldn't hurt. If you lose this key, you can't decrypt secrets encrypted
with it.

Secrets are named after the environment variable they produce, so `db_dsn`,
`db-dsn` and `DB_DSN` are the same secret and `create` updates the one that
is already there. A name which can't produce a usable environment variable
name, like `1foo`, is rejected.

A secret is a tuple of [ID, Name, Value]. The Value is the only encrypted
field within the JSON document. The ID is a ULID, so it sorts by creation
time. It currently isn't used, but it may be used in the future, within
some kind of service that provides an API to synchronize secrets between
repositories and some kind of central management plane for your
infrastructure. Files written by earlier versions hold a UUID in that
field; those are read as they are and keep their ID.

## Example usage


Adding a secret:

~~~
# ./secretsmanager create
Name for your new secret: DB_DSN
Secret value: user:password@hostname
Created new secret:

ID: 01M0PYDT91YCR2GB82ANMG28EB
Name: DB_DSN
Value: user:password@hostname
~~~

The `.secrets.json` contents:

~~~
# cat .secrets.json
{
  "secrets": [
    {
      "ID": "01M0PYDT91YCR2GB82ANMG28EB",
      "Name": "DB_DSN",
      "Value": "0mT85-AsvIYeyfNkjp_PE3dNoqIHCzO1NQHg-Y3iECoY4Y6DLY4="
    }
  ]
}
~~~

Generating secrets for environment usage:

~~~
# ./secretsmanager env
DB_DSN="user:password@hostname"
~~~

For this particular case, you'd use `secretsmanager env >> .env` to
produce the secrets as additional environment variables. Values are
quoted for a shell, so a value with a `$`, a backtick or a backslash in
it comes back out of `.env` unchanged.

Reading a single secret in a script:

~~~
# ./secretsmanager get DB_DSN
user:password@hostname

# DSN=$(secretsmanager get DB_DSN)
~~~

`get` prints the value and nothing else, and exits non-zero if there is
no such secret. Together with `SECRETSMANAGER_WORKSPACE` this works from
any directory.

## Use from Go

The same storage the cli runs on is importable, so a program can read its
own secrets without shelling out:

~~~go
package main

import (
	"context"
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/titpetric/secretsmanager"
)

func main() {
	ctx := context.Background()

	secrets, err := secretsmanager.NewStorage(secretsmanager.NewOptionsFromEnv())
	if err != nil {
		log.Fatal(err)
	}

	dsn, err := secrets.Get(ctx, "DB_DSN")
	if err != nil {
		// errors.Is(err, secretsmanager.ErrNotFound) when it isn't stored
		log.Fatal(err)
	}

	db, err := sql.Open("mysql", dsn.Value)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
}
~~~

`NewOptionsFromEnv` reads `SECRETSMANAGER_WORKSPACE` and
`SECRETSMANAGER_KEY`. Pass the `Options` yourself to keep the key out of
the environment, or to serve two workspaces with two keys at once:

~~~go
secrets, err := secretsmanager.NewStorage(secretsmanager.Options{
	Workspace: "/srv/app",
	Key:       key, // 32 bytes, as init generates
})
~~~

Nothing lives in package state, so each storage encrypts with the key it
was given. `docs/api.md` lists the whole surface.

`Storage` is an interface, so a driver other than the local file can be
substituted later:

~~~go
type Storage interface {
    fmt.Stringer

    List(ctx context.Context) ([]*Secret, error)
    Get(ctx context.Context, name string) (*Secret, error)
    Set(ctx context.Context, name, value string) (*Secret, error)
}
~~~

`Get` and `List` never write. `Set` writes one secret and leaves the
ciphertext of the others as it found it. That interface, `Options`,
`Secret` and `ErrNotFound` are the whole package, and `docs/api.md` lists
it.

## Closing notes

Less is more.