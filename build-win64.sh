CGO_ENABLED=1 GOOS=windows GOARCH=amd64 PKG_CONFIG_LIBDIR=/usr/x86_64-w64-mingw32/sys-root/mingw/lib/pkgconfig CXX=x86_64-w64-mingw32-g++ CC=x86_64-w64-mingw32-gcc go build -o bin/rebot.exe rebot/main.go
x86_64-w64-mingw32-strip bin/rebot.exe
cp /usr/x86_64-w64-mingw32/sys-root/mingw/bin/SDL2.dll bin
