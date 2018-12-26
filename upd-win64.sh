#! /bin/sh

CGO_ENABLED=1 GOOS=windows GOARCH=amd64 PKG_CONFIG_LIBDIR=/usr/x86_64-w64-mingw32/sys-root/mingw/lib/pkgconfig:/usr/lib/pkgconfig:/usr/share/pkgconfig CXX=x86_64-w64-mingw32-g++ CC=x86_64-w64-mingw32-gcc go get -u github.com/kjx98/go-sdl2/sdl
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 PKG_CONFIG_LIBDIR=/usr/x86_64-w64-mingw32/sys-root/mingw/lib/pkgconfig:/usr/lib/pkgconfig:/usr/share/pkgconfig CXX=x86_64-w64-mingw32-g++ CC=x86_64-w64-mingw32-gcc go get -u github.com/kjx98/golib/to
