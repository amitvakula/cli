#!/usr/bin/env bash
set -e
set -o pipefail
unset CDPATH; cd "$( dirname "${BASH_SOURCE[0]}" )"

pkg="flywheel.io/scientist"       # Project's full package name
goVersion=${GO_VERSION:-"1.6.1"}  # Which go version to use by default
goIsolate=false                   # Allow gimme to install a user-wide go install?

# Isolate this project's gopath
export GOPATH="$PWD"/.gopath/

getSub() {
	git submodule update --init $1 | (grep -v ': checked out ' || true)
}

runGimme() { # Provision a Go environment.
	test -z $SKIP_GIMME || return 0
	folder=".gopath/src/github.com/travis-ci/gimme"
	export gimme="$folder/gimme"

	test -f $gimme || getSub $folder # Acquire gimme
	export GIMME_GO_VERSION=$goVersion # Use project's go version
	export GIMME_SILENT_ENV=1 # Don't make noise on launch
	export GIMME_TYPE="binary" # Don't run off into the trees and invoke make.bash :|
	gFolder="./.gimme" # Would otherwise use /tmp/gimme; so much wrong with that
	export GIMME_TMP="$gFolder-tmp" # Use a local folder instead!

	if $goIsolate; then
		# Isolate the go install inside this project, rather than using homedir
		export GIMME_VERSION_PREFIX="$gFolder/versions"
		export GIMME_ENV_PREFIX="$gFolder/envs"
		export GIMME_TMP="$gFolder/tmp"
	fi

	filterActions='/^+/ {/^+ tar/p; /^+ curl/p; d};' # Only show curl & tar
	filterEnv='/^unset/d; /^export/d;' # Filter out env setting (easier than eval)
	filterEmpty='/^\s*$/d;' # Filter out empty lines
	filterBinary='/(using type .*)/d;' # Filter out superfluous binary type msg
	filterError='s/'"I don't have any idea what to do with"'/Download or install failed for go/g;' # On error, print a message that actually makes sense

	# Install go, clearing tempdir before & after, then load env variables
	rm -rf $GIMME_TMP
	GIMME_DEBUG=1 $gimme 2>&1 | sed "$filterActions $filterEnv $filterEmpty $filterBinary $filterError"
	rm -rf $GIMME_TMP
	eval "$($gimme)" # Was easier to invoke than eval. Improvements accepted
}

prepGlide() {
	glideHashFile=".gopath/.glidehash"
	folder=".gopath/src/github.com/Masterminds/glide"
	glide="$folder/glide"

	test -f $glide || ( # Build glide from source if neccesary
		getSub $folder; cd $folder
		version=`git describe --tags` # Remove need for makefile by running manually
		go build -v -o glide -ldflags "-X main.version=$version" glide.go # Build
	)
}

runGlide() {
	test -z $SKIP_GLIDE || return 0
	prepGlide

	genHash() { # Cache glide install runs by hashing state
		cat glide.lock glide.yaml | sha1sum | cut -f 1 -d ' '
	}

	install() { # Whenever glide runs, update the hash marker
		$glide install
		genHash > $glideHashFile
	}

	# If glide components are missing, or cache is out of date, run
	test -f glide.lock -a -d vendor -a -f $glideHashFile || install
	test `cat $glideHashFile` = `genHash` || install

}

novendor() {
	# Glide novendor is good at finding folders with go code, but finds the local folder (.) too!
	a=(`./make.sh glide novendor -x | grep -v '^.$'`)
	b=(`find -maxdepth 1 -type f -name '*.go'`)
	echo ${a[@]} ${b[@]}
}

format() {
	gofmt -w $(novendor)
}
formatCheck() {
	badFiles=(`gofmt -l $(novendor)`)
	if [[ ${#badFiles[@]} -gt 0 ]]; then
		echo "The following files need formatting: " ${badFiles[@]}
		exit 1
	fi
}

build() {
	# Go's output is incredibly messy; force some sanity by removing filepaths
	cleanVendor='s#'$PWD'/.gopath/src/'$pkg'/vendor/#vendor/#g;' # Import lookups
	cleanVendor2='s#'$pkg'/vendor/##g;' # Error lines
	cleanPwd='s#'$PWD'/.gopath/src/#$GOPATH/#g;' # Import lookups
	cleanRoot='s#'$GOROOT'/src/#$GOROOT/#g;' # Import lookups
	cleanErrors='s$^.gopath/src/$$g;' # General shenanigans
	matchGo='[a-zA-Z0-9_]*\.go|' # Match go source files

	# Sed & grep clean up paths and highlights the name of offending files, respectively
	go install -v $pkg 2>&1 | sed -u "$cleanVendor $cleanVendor2 $cleanPwd $cleanRoot $cleanErrors" | (egrep --color "$matchGo" || true)

}

run() {
	test -f .gopath/bin/panicparse || go install -v github.com/maruel/panicparse
	.gopath/bin/`basename $pkg` ${args[@]} 2> >( .gopath/bin/panicparse )
	sleep 0.1; echo
}

_test() {
	./make.sh glide novendor -x | sed 's#^\.#'$pkg'#g' | xargs ./make.sh go test
}

prep() {
	runGimme; runGlide
}


args=("$@")
args[0]=${args[0]:-"build"}
cmd=${args[0]}
args=("${args[@]:1}")

case "$cmd" in
	"build") # Standard full build (the default)
		prep
		build
		format
		;;
	"make") # Just compile
		prep
		build
		;;
	"format") # Format in-place
		runGimme
		format
		;;
	"test") # Unit tests
		prep
		_test
		;;
	"run") # Execute binary with parse util
		prep
		build
		run
		;;
	"novendor") # Output source folders & files
		novendor
		;;
	"go" | "godoc" | "gofmt") # Go commands in project context
		runGimme
		$cmd ${args[@]}
		;;
	"glide") # Glide commands in project context
		runGimme
		prepGlide
		$glide ${args[@]}
		;;
	"ci") # Monolithic CI target
		prep
		build
		formatCheck
		;;
	*)
		echo "Usage: ./make.sh {build|make|format|test|run|novendor|go|godoc|gofmt|glide|ci}"
		exit 1
		;;
esac
