Jot
===

Jot is a simple and editable pastebin that's super easy to host yourself. It
doesn't require installation of large dependencies and everything runs from a
single binary. Outside a data dir and encryption seed file for password
generation, you don't need a programming language installed or assets anywhere
on your server.

## Features

The Jot feature set is very simple. The server exposes a very limited set of
CRUD operations defined below:

- `POST` content and get back a unique URL (`JOT_URL`) and a Jot-Password header
- `GET` to `JOT_URL` and you get back the content and an ETag header with the last
  modified date set.
- `GET` to `JOT_URL` with `If-None-Match` header set and you will get a 304 Not
  Modified status code back with no content, if the last modified date matches
  the value sent in that header. Useful for client caching. Browsers support
  this out of the box.
- `PUT` to `JOT_URL` with `?password=<Jot-Password value>` and you can update that
  content.
- `PUT` supports `If-Match` header allowing you to bail if the content has been
  updated since the last modified date returned with a `GET`.
- `DELETE` to `JOT_URL` with `?password=<Jot-Password value>` and you will delete
  the jot.

## Endpoints

`GET /`: help text

`POST /txt`: create a text jot

`GET /txt/<id>`: get a text jot

`PUT /txt/<id>?password=<password>`: edit a text jot

`DELETE /txt/<id>?password=<password>`: delete a text jot

`POST /img`: upload an image

`GET /img/<id>`: get an image

`DELETE /img/<id>?password=<password>`: delete an image

## Building and Running

Requires: Go >=1.14

Running from localhost is simple. Below describes the minimal effort that goes
into running Jot:

```
# setup the source code dir
export JOT_HOME="${HOME}/code/jot"
mkdir -p "${JOT_HOME}"
git clone https://github.com/kyleterry/jot "${JOT_HOME}"

# these two are for config and data storage
mkdir -p "${HOME}/.config/jot"
mkdir -p "${HOME}/.local/share/jot"

# export the configuration for the jot proc
export JOT_MASTER_PASSWORD="please change this, this is your master password (also don't lose it)"
export JOT_SEED_FILE="${HOME}/.config/jot/seed"
export JOT_DATA_DIR="${HOME}/.local/share/jot"

cd "${JOT_HOME}"

# generate the seed file encrypted with the master password
go run ./vendor/github.com/cloudflare/gokey/cmd/gokey -p "${JOT_MASTER_PASSWORD}" -t seed -o "${JOT_SEED_FILE}"

# build and run jot
go build ./cmd/jot

./jot

curl http://localhost:8095
```
