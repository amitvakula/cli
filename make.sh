#!/usr/bin/env bash
set -e
set -o pipefail
unset CDPATH; cd "$( dirname "${BASH_SOURCE[0]}" )"; cd "`pwd -P`"

pkg="flywheel.io/fw"       # Project's full package name
goV=${GO_VERSION:-"1.7.1"} # Project's default Go version
minGlideV="0.12.3"         # Project's minimum Glide version

# Load GNU coreutils on OSX
if [[ "$(uname -s)" == "Darwin" ]]; then
	which brew gsort gsed flock > /dev/null || (
		echo "On OSX, homebrew is required. Install from http://brew.sh"
		echo "Then, run 'brew install coreutils gnu-sed flock' to install the necessary tools."
	)
	export PATH="$(brew --prefix coreutils)/libexec/gnubin:$PATH"
	export PATH="$(brew --prefix gnu-sed)/libexec/gnubin:$PATH"
fi

prepareGo() {
	export GOPATH=$(cd ../../../; pwd) # Isolate this project's gopath
	export GIMME_GO_VERSION=$goV       # Specify which Go version to install
	export GIMME_SILENT_ENV=1          # Suppress logging on launch
	export GIMME_TYPE="binary"         # Only use binary distributions
	export GIMME_TMP="./.gimme-tmp"    # Would otherwise use /tmp/gimme, which is fragile
	export GIMME_DEBUG=1               # Print installation commands (shows progress to user)
	src=~/.gimme/envs/go${goV}.env     # Locate the source env file

	# Clean up gimme's logging
	filterActions='/^+/ {/^+ tar/p; /^+ curl/p; d};' # Only show curl & tar
	filterEnv='/^unset/d; /^export/d;'               # Filter env setting
	filterEmpty='/^\s*$/d;'                          # Filter empty lines
	filterBinary='/(using type .*)/d;'               # Filter distribution type message
	filterError='s/'"I don't have any idea what to do with"'/Download or install failed for go/g;' # Helpful error message

	# Install go, clearing tempdir before & after, with nice messaging.
	test -f $src || (
		echo "Downloading go $goV..."
		rm -rf $GIMME_TMP
		curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | bash 2>&1 | sed "$filterActions $filterEnv $filterEmpty $filterBinary $filterError"
		rm -rf $GIMME_TMP
	)

	# Load installed go and prepare for compiled tools
	source $src
	export PATH=$GOPATH/bin:$PATH
}

installGlide() {
	echo "Downloading glide $minGlideV or higher..."
	mkdir -p $GOPATH/bin
	rm -f $GOPATH/bin/glide
	curl -sL https://glide.sh/get | bash
}

cleanGlide() {
	# Timestamps confuse the diff, and glide works fine without this field
	sed -i '/^updated: /d' glide.lock
}

prepareGlide() {
	test -x $GOPATH/bin/glide || installGlide

	# Check the current glide version against the minimum project version
	currentVersion=$(glide --version | cut -f 3 -d ' ' | tr -d 'v')
	floorVersion=$(echo -e "$minGlideV\n$currentVersion" | sort -V | head -n 1)

	if [[ $minGlideV != $floorVersion ]]; then
		echo "Glide $currentVersion is older than required minimum $minGlideV; upgrading..."
		installGlide
	fi

	# Cache glide install runs by hashing state
	glideHashFile=".glidehash"
	genHash() {
		cleanGlide
		# Using `shasum` because `sha1sum` does not exist on OSX by default.
		cat glide.lock glide.yaml | shasum -a 1 | cut -f 1 -d ' '
	}

	install() {
		# Whenever glide runs, update the hash marker
		glide install
		genHash > $glideHashFile
	}

	# If glide components are missing, or cache is out of date, run
	test -f glide.lock -a -d vendor -a -f $glideHashFile || install
	test `cat $glideHashFile` = `genHash` || install
}

build() {
	# Go prints lot of absolute paths; unclutter them.
	hideGoroot="s#$GOROOT/src/##g;"
	hideGopath="s#$GOPATH/src/##g;"
	hideVendor="s#$pkg/vendor/##g;"
	hidePWD="s#$PWD/##g;"
	matchGo='[a-zA-Z0-9_]*\.go|'     # Match go source files

	# Go install uses $GOPATH/pkg to cache built files
	# Sed to clean up logging, then egrep to highlight parts of the output
	go install -v -ldflags '-s' $pkg 2>&1 | sed -u "$hideGoroot $hideGopath $hideVendor $hidePWD" | (egrep --color "$matchGo" || true)
}

crossBuild() {
	oses=( "darwin" "linux" "windows" )
	arches=( "386" "amd64" )
	mkdir -p release

	cross() {
		echo
		echo "-- Building $os $arch --"

		binary="release/`basename $pkg`-$os-$arch"

		env GOOS=$os GOARCH=$arch nice go build -v -ldflags '-s' -o $binary $pkg
		which upx > /dev/null && nice upx -q $binary 2>&1 | grep -C 1 -- "---" || true
	}

	# Run builds in parallel, displaying output sequentially
	pids=()
	for os in "${oses[@]}"; do
		for arch in "${arches[@]}"; do
			( cross 2>&1 | flock release/.lock cat ) &
			pids+=($!)
		done
	done

	fail=true
	for pid in "${pids[@]}"; do wait $pid || fail=false; done
	rm -f release/.lock cat
	if [ $fail = false ]; then echo; echo "Compilation failed"; exit 1; fi

	echo
	echo "Cross-compilation success!"
	which upx > /dev/null || ( echo "UPX is not installed; not packing binaries." )
}

# List packages with source code: packageA, packageB... First argument is a prefix (optional)
listPackages() {
	find . -name '*.go' -printf "%h\n" | sed -E '/vendor/d; s#^\./##g; /\./d; s#^#'$1'#g' | sort | uniq
}

# List files in the main package. First argument is a prefix (optional)
listBaseFiles() {
	find -maxdepth 1 -type f -name '*.go' | sed 's#^\./##g; s#^#'$1'#g'
}

format() {
	gofmt -w -s $(listPackages; listBaseFiles)
}

formatCheck() {
	badFiles=(`gofmt -l -s $(listPackages; listBaseFiles)`)
	if [[ ${#badFiles[@]} -gt 0 ]]; then
		echo "The following files need formatting: " ${badFiles[@]}
		exit 1
	fi
}

_test() {
	hideEmptyTests="/\[no test files\]$/d;"
	go test -v "$@" $pkg $(listPackages $pkg/) | sed "$hideEmptyTests"
}

args=("$@")
args[0]=${args[0]:-"build"}
cmd=${args[0]}
args=("${args[@]:1}")

case "$cmd" in
	"go" | "godoc" | "gofmt") # Go commands in project context
		prepareGo
		$cmd ${args[@]}
		;;
	"glide") # Glide commands in project context
		prepareGo
		prepareGlide
		glide ${args[@]}
		cleanGlide
		;;
	"make") # Just build
		prepareGo
		prepareGlide
		build
		;;
	"build") # Full build (the default)
		prepareGo
		prepareGlide
		build
		format
		;;
	"format") # Format code
		prepareGo
		format
		;;
	"clean") # Remove build state
		prepareGo
		rm -rvf $GOPATH/pkg $GOPATH/bin/${pkg##*/}
		;;
	"test") # Run tests
		prepareGo
		prepareGlide
		_test ${args[@]}
		;;
	"env") # Load environment!   eval $(./make.sh env)
		prepareGo 1>&2
		prepareGlide 1>&2
		(go env; echo "PATH=$PATH") | sed 's/^/export /g'
		;;
	"ci") # Monolithic CI target
		prepareGo
		prepareGlide
		build
		formatCheck
		_test -race
		;;
	"cross") # Cross-compile to every platform
		prepareGo
		prepareGlide
		build
		crossBuild
		;;
	*)
		echo "Usage: ./make.sh {}"
		exit 1
		;;
esac
