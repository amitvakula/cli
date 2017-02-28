#!/usr/bin/env bash
set -euo pipefail
unset CDPATH; cd "$( dirname "${BASH_SOURCE[0]}" )"; cd "`pwd -P`"

# Project settings: package name, Go & Glide versions, and cross-compilation targets
pkg="flywheel.io/fw"
goV=${GO_VERSION:-"1.8"}
minGlideV="0.12.3"
targets=( "darwin/amd64" "linux/amd64" "windows/amd64" )
#

fatal() {
	echo -e $1; exit 1
}

# Check that this project is in a gopath
test -d ../../../src || fatal "This project must be located in a gopath.\nTry cloning instead to \"src/$pkg\"."
export GOPATH=$(cd ../../../; pwd)

# Get system info
localOs=$( uname -s | tr '[:upper:]' '[:lower:]' )
localArch=$( uname -m | sed 's/x86_//; s/i[3-6]86/32/' )

# Load GNU coreutils on OSX
if [[ "$localOs" == "darwin" ]]; then
	# Check requirements: g-prefixed commands are available if brew packages are installed.
	which brew gsort gsed gfind > /dev/null || fatal "On OSX, homebrew is required. Install from http://brew.sh\nThen, run 'brew install bash coreutils findutils gnu-sed' to install the necessary tools."

	# Load GNU coreutils, findutils, and sed into path
	suffix="libexec/gnubin"
	export PATH="$(brew --prefix coreutils)/$suffix:$(brew --prefix findutils)/$suffix:$(brew --prefix gnu-sed)/$suffix:$PATH"

	# OSX has shasum. CentOS has sha1sum. Ubuntu has both.
	alias sha1sum="shasum -a 1"
fi

prepareGo() {
	# Configure gimme: get our desired Go version with reasonable logging, only binary downloads, and local state folder
	export GIMME_GO_VERSION=$goV; export GIMME_SILENT_ENV=1; export GIMME_DEBUG=1
	export GIMME_TYPE="binary"; export GIMME_TMP="./.gimme-tmp"

	# Inherit or set the source directory
	: "${GIMME_ENV_PREFIX:=${HOME}/.gimme/envs}"
	src="${GIMME_ENV_PREFIX}/go${goV}.env"

	# Show download & extract progress, removing other commands, empty lines, and rewrite error message
	filterLog='/^\+ (curl|wget|fetch|tar|unzip)/p; /^\++ /d; /^(unset|export) /d; /(using type .*)/d; /^$/d;'
	filterError='s/'"I don't have any idea what to do with"'/Download or install failed for go/g;'

	# Install go, clearing tempdir before & after, with nice messaging.
	test -f $src || (
		echo "Downloading go $goV..."
		rm -rf $GIMME_TMP; mkdir -p $GIMME_TMP

		curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme > $GIMME_TMP/gimme.sh
		chmod +x $GIMME_TMP/gimme.sh

		$GIMME_TMP/gimme.sh 2>&1 | sed -r "$filterLog $filterError"
		rm -rf $GIMME_TMP
	)

	# Load installed go and prepare for compiled tools
	source $src
	export PATH=$GOPATH/bin:$PATH
}

cleanGlideLockfile() {
	# Remove timestamp and hash from glide lockfiles. Pollutes diff, and does not prevent normal operation.
	sed -i '/^updated: /d; /^hash: /d' glide.lock
}

prepareGlide() {
	prepareGo

	# Cache glide install runs by hashing state
	glideHashFile=".glidehash"

	installGlide() {
		echo "Downloading glide $minGlideV or higher..."
		mkdir -p $GOPATH/bin
		rm -f $GOPATH/bin/glide
		curl -sL https://glide.sh/get | bash
	}

	generateGlideHash() {
		cleanGlideLockfile
		cat glide.lock glide.yaml | sha1sum | cut -f 1 -d ' '
	}

	runGlideInstall() {
		# Whenever glide runs, update the hash marker
		glide install
		generateGlideHash > $glideHashFile
	}

	test -x $GOPATH/bin/glide || installGlide

	# Check the current glide version against the minimum project version
	currentVersion=$(glide --version | cut -f 3 -d ' ' | tr -d 'v')
	floorVersion=$(echo -e "$minGlideV\n$currentVersion" | sort -V | head -n 1)

	if [[ $minGlideV != $floorVersion ]]; then
		echo "Glide $currentVersion is older than required minimum $minGlideV; upgrading..."
		installGlide
	fi

	# If glide components are missing, or cache is out of date, run
	test -f glide.lock -a -d vendor -a -f $glideHashFile || runGlideInstall
	test `cat $glideHashFile` = `generateGlideHash` || runGlideInstall
}

build() {
	# Clean out the absolute-path pollution, and highlight source code filenames.
	filterAbsolutePaths="s#$GOROOT/src/##g; s#$GOPATH/src/##g; s#$pkg/vendor/##g; s#$PWD/##g;"
	highlightGoFiles="s#[a-zA-Z0-9_]*\.go#$(tput setaf 1)&$(tput sgr0)#;"

	# Go install uses $GOPATH/pkg to cache built files. Faster than go build.
	#
	# Adding -ldflags '-s' strips the the DWARF symbol table and debug information.
	# https://golang.org/cmd/link
	#
	# One downside to this is that cross-compiling will require $GOROOT be writeable.
	# This seems an entirely reasonable tradeoff: the alternative is *recompiling the standard library N times every build*.
	# As luck would have it, we're using gimme anyway, which provides a safe, writable Go install in your homedir.
	# https://dave.cheney.net/2015/08/22/cross-compilation-with-go-1-5
	go install -v -ldflags '-s' $pkg 2>&1 | sed "$filterAbsolutePaths $highlightGoFiles"
}

crossBuild() {
	for target in "${targets[@]}"; do

		# Split target on slash to get operating system & architecture
		IFS='/'; targetSplit=($target); unset IFS;
		os=${targetSplit[0]}; arch=${targetSplit[1]}

		echo -e "\n-- Building $os $arch --"
		GOOS=$os GOARCH=$arch build

		# Versions of UPX prior to 3.92 had compatibility issues with OSX 10.12 Sierra.
		# Could augment this with a UPX version check, and compress if local UPX is modern enough.
		#
		# https://upx.github.io/upx-news.txt
		# https://apple.stackexchange.com/questions/251808/this-upx-compressed-binary-contains-an-invalid-mach-o-header-and-cannot-be-load
		if [[ "$os" != "darwin" ]]; then
			if [[ "$localOs" == "$os" && "$arch" =~ .*$localArch ]] ; then
				path="$GOPATH/bin/"
			else
				path="$GOPATH/bin/${os}_${arch}"
			fi

			binary=$( find "$path" -maxdepth 1 | grep -E "${pkg##*/}(\.exe)*" | head -n 1 )

			which upx > /dev/null && nice upx -q $binary 2>&1 | grep -- "->" || true
		fi
	done

	which upx > /dev/null || ( echo "UPX is not installed; did not compress binaries." )
}

# Some go tools take package names. Some take file names. Some like the pacakge prefix. Some don't.
# None of them omit the vendor directory like they should.
# What follows is a result of these problems.
# https://github.com/golang/go/issues/19090


# List packages with source code: packageA, packageB... First argument is a prefix (optional)
listPackages() {
	prefix=${1:-}
	find . -name '*.go' -printf "%h\n" | sed '/^\.\/vendor/d; s#^\./##g; /\./d; s#^#'$prefix'#g' | sort --unique
}

# List files in the main package. First argument is a prefix (optional)
listBaseFiles() {
	prefix=${1:-}
	find -maxdepth 1 -type f -name '*.go' | sed 's#^\./##g; s#^#'$prefix'#g'
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


cmd=${1:-"build"}
shift || true

case "$cmd" in
	"go" | "godoc" | "gofmt") # Go commands in project context
		prepareGo
		$cmd "$@"
		;;
	"glide") # Glide commands in project context
		prepareGlide
		glide "$@"
		cleanGlideLockfile
		;;
	"make") # Just build
		prepareGlide
		build
		;;
	"build") # Full build (the default)
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
		prepareGlide
		_test "$@"
		;;
	"env") # Load environment!   eval $(./make.sh env)
		prepareGlide 1>&2
		(go env; echo "PATH=$PATH") | sed 's/^/export /g'
		;;
	"ci") # Monolithic CI target
		prepareGlide
		build
		formatCheck
		_test -race
		;;
	"cross") # Cross-compile to every platform
		prepareGlide
		crossBuild
		;;
	*)
		echo "Usage: ./make.sh {go|godoc|gofmt|glide|make|build|format|clean|test|env|ci|cross}"
		exit 1
		;;
esac
