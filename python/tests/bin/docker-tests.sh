#!/usr/bin/env sh

set -eu
unset CDPATH
cd "$( dirname "$0" )/../.."


USAGE="
Usage:
    $0 [OPTION...] [[--] TEST_ARGS...]

Build test image and run tests in a Docker container.

Options:
    -h, --help          Print this help and exit

    -B, --no-build      Skip rebuilding default Docker image
        --image IMAGE   Use custom Docker image

    TEST_ARGS           Arguments passed to tests.sh

"

main() {
    local DOCKER_IMAGE=
    local RUN_SHELL=

    while [ $# -gt 0 ]; do
        case "$1" in
            -h|--help)
                log "$USAGE"
                exit 0
                ;;
            -B|--no-build)
                DOCKER_IMAGE="flywheel_cli:testing"
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

    if [ -z "${DOCKER_IMAGE}" ]; then
        log "Building flywheel_cli:testing ..."
        docker build --tag flywheel_cli:testing .
    fi

    trap clean_up EXIT

    docker run -it \
        --name flywheel-cli-test \
        --volume $(pwd):/app/src \
        --workdir /app/src \
        flywheel_cli:testing \
        tests/bin/tests.sh "$@"
}

clean_up() {
    local TEST_RESULT_CODE=$?
    set +e

    log "INFO: Cleaning up container(s)..."
    docker rm --force --volumes flywheel-cli-test

    [ "$TEST_RESULT_CODE" = "0" ] && log "INFO: Test return code = $TEST_RESULT_CODE" \
                                  || log "ERROR: Test return code = $TEST_RESULT_CODE"

    exit $TEST_RESULT_CODE
}

log() {
    printf "\n%s\n" "$@" >&2
}

main "$@"
