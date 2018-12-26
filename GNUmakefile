#
#	Makefile for hookAPI
#
# switches:
#	define the ones you want in the CFLAGS definition...
#
#	TRACE		- turn on tracing/debugging code
#
#
#
#

# Version for distribution
VER=1_0r1
GOPATH=$(shell go env GOPATH):$(PWD)

export GOPATH
MAKEFILE=GNUmakefile

# We Use Compact Memory Model

all: bin/rebot
	@[ -d bin ] || exit

bin/rebot: rebot/main.go
	@[ -d bin ] || mkdir bin
	go build -o bin/rebot rebot/main.go
	@strip $@ || echo "rebot OK"

win64: bin/rebot.exe

bin/rebot.exe: bin rebot/main.go
	@./build-win64.sh

clean:

distclean: clean
	@rm -rf bin
