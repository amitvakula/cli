#!/usr/bin/env sh

set -eu
unset CDPATH
cd "$( dirname "$0" )/../.."


USAGE="
Usage:
    $0 [OPTION...] [[--] PYTEST_ARGS...]

Runs all tests (unit, integ and linting) if no options are provided.

Assumes running in a flywheel/core:testing container or that core and all
of its dependencies are installed the same way as in the Dockerfile.

Options: -h, --help           Print this help and exit -s, --shell          Enter shell instead of running tests PYTEST_ARGS          Arguments passed to py.test
"

# Execute the tests
main() {
    export PYTHONDONTWRITEBYTECODE=1
    local RUN_SHELL=false

    while [ $# -gt 0 ]; do
        case "$1" in
            -h|--help)
                log "$USAGE"
                exit 0
                ;;
            -S|--shell)
                RUN_SHELL=true
                ;;
            --)
                shift
                break
                ;;
            *)
                break
                ;;
        esac
        shift
    done

    # Start postgres
    start_postgresql
    export POSTGRES_TEST_DB="dbname=ingest_db user=ingest password=password"

    if [ $RUN_SHELL = true ]; then
        log "INFO: Entering test shell ..."
        sh
        exit
    fi

    py.test tests/unit_tests/  "$@"
}

start_postgresql() {
    log "Starting postgres"
    su postgres -c "postgres -D /db/postgres >/var/log/postgresql/console.log 2>&1 &"

    log "INFO: Waiting for postgres..."
    until su postgres -c "pg_isready -q"; do
        sleep 0.1
    done
    log "INFO: Postgres is up!"
    su postgres -c "psql -c \"CREATE ROLE ingest WITH LOGIN PASSWORD 'password'\""
    su postgres -c "psql -c \"CREATE DATABASE ingest_db\""
    su postgres -c "psql -c \"GRANT ALL PRIVILEGES ON DATABASE ingest_db TO ingest\""
}

log() {
    printf "\n%s\n" "$@" >&2
}

main "$@"
