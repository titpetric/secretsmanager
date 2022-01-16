# secretsmanager

I created secretsmanager to store some secrets within a repository. The
secrets are encrypted at rest, with readable keys and editable JSON, so
you can rename a key or delete it by hand. The cli tool handles the bare
minumum of requirements.

- `secretsmanager init` - generate the encryption key,
- `secretsmanager create` - create a new secret,
- `secretsmanager env` - print secrets for .env

The tool will modify `.secrets.json` as needed.

The encryption key generated with `init` should not be commited into git.
It should be added to the ambient environment on your system, or your
deployment / CI pipeline. If you want to store it into 1password it also
wouldn't hurt. If you lose this key, you can't decrypt secrets encrypted
with it.

A secret is a tuple of [UUID, Name, Value]. The Value is the only
encrypted field within the JSON document. The UUID field currently isn't
used, but it may be used in the future, within some kind of service that
provides an API to synchronize secrets between repositories and some kind
of central management plane for your infrastructure.

## Example usage


Adding a secret:

~~~
# ./secretsmanager create
Name for your new secret: DB_DSN
Secret value: user:password@hostname
Created new secret:

ID: 25349927-99b2-4ac5-ad59-d63f88f4a612
Name: DB_DSN
Value: user:password@hostname
~~~

The `.secrets.json` contents:

~~~
# cat .secrets.json
{
  "secrets": [
    {
      "ID": "25349927-99b2-4ac5-ad59-d63f88f4a612",
      "Name": "DB_DSN",
      "Value": "NkFdj_eaROsyRDplbGj0mupw0CTLpHWemjE3N3ktvs-Fwv2lJQw="
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
produce the secrets as additional environment variables.

## Closing notes

Less is more.