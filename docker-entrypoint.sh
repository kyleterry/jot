#!/bin/bash

set -Eeo pipefail

export JOT_DATA_DIR="${JOT_DATA_DIR:-/var/lib/jot}"
export JOT_SEED_FILE="${JOT_SEED_FILE:-/etc/jot/seed}"
export JOT_BIND_ADDR="0.0.0.0:8095"

if [ -z "${JOT_MASTER_PASSWORD}" ]; then
    echo >&2 "error: JOT_MASTER_PASSWORD not set"
    exit 1
fi

if [ ! -f ${JOT_SEED_FILE} ]; then
    gokey -p "${JOT_MASTER_PASSWORD}" -t seed -o "${JOT_SEED_FILE}"
fi

exec $@
