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

bin/rebot: rebot/main.go sdl2.go config.go wecat.go
	@[ -d bin ] || mkdir bin
	go build -o bin/rebot rebot/main.go
	@strip $@ || echo "rebot OK"

win64: bin/rebotWin64.zip

bin/rebot.exe: rebot/main.go sdl2.go config.go wecat.go
	(. ./mingw64-env.sh; go build -o $@ rebot/main.go)
	@strip $@ || echo "rebot.exe win64 OK"

bin/rebotWin64.zip: bin/rebot.exe bin/SDL2.dll
	@(cd bin; zip rebotWin64 rebot.exe SDL2.dll)

bin/SDL2.dll:
	@cp /usr/x86_64-w64-mingw32/sys-root/mingw/bin/SDL2.dll bin

clean:

distclean: clean
	@rm -rf bin
