Jot
===

Jot is a simple and editable pastebin that's super easy to host yourself. It
doesn't require installation of large dependancies and everything runs from a
single binary. Outside a data dir and encryption seed file for password
generation, you don't need a programming language installed or assets anywhere
on your server.

Making things even easier is a docker image that will create your seedfile for
you and store it in a volume mount.

Jot is meant to be used heavily on the command line or within programs, but you
can also use it to quickly share a text file (like code or configuration) with a
friend. There is no web interface outside a help manual returned from a `GET`
request to `/`.

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

## Building and installing

Building jot requires Go 1.10 (but no vendoring is included) or Go 1.11 with
module support enabled.

A simple `make` will build jot, storing a resulting binary in `bin`

You can build the docker image with `make build-docker` which will yield a
container tagged `kyleterry/jot:latest`.

## Running

Running from localhost is simple. Below describes the minimal effort that goes
into running Jot:

```
mkdir -p ${HOME}/.config/jot
mkdir -p ${HOME}/.local/share/jot
mkdir -p ${HOME}/src

export JOT_SEED_FILE=${HOME}/.config/jot/seed
export JOT_DATA_DIR=${HOME}/.local/share/jot
export JOT_MASTER_PASSWORD="please change this, this is your master password (also don't lose it)"

cd ${HOME}/src
git clone https://github.com/kyleterry/jot
cd jot
GO111MODULE=on make
./bin/jot

curl http://localhost:8095
```
