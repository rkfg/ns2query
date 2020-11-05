#!/bin/sh
cd $(dirname "$0")
PROJECT='ns2query'
export CGO_ENABLED=0
rm -rf release
mkdir release
rm -rf tmp
mkdir tmp
cd tmp
GOARCH=mipsle go build -ldflags='-s -w -extldflags -static' -o $PROJECT.mipsel ..
7z a ../release/$PROJECT.mipsel.zip $PROJECT.mipsel
GOARCH=arm go build -ldflags='-s -w -extldflags -static' -o $PROJECT.arm ..
7z a ../release/$PROJECT.arm.zip $PROJECT.arm
GOARCH=386 go build -ldflags='-s -w -extldflags -static' -o $PROJECT.386 ..
7z a ../release/$PROJECT.linux32.zip $PROJECT.386
GOARCH=amd64 go build -ldflags='-s -w -extldflags -static' -o $PROJECT.amd64 ..
7z a ../release/$PROJECT.linux64.zip $PROJECT.amd64
GOOS=windows GOARCH=386 go build -ldflags='-s -w -extldflags -static' -o $PROJECT.32.exe ..
7z a ../release/$PROJECT.win32.zip $PROJECT.32.exe
GOOS=windows GOARCH=amd64 go build -ldflags='-s -w -extldflags -static' -o $PROJECT.64.exe ..
7z a ../release/$PROJECT.win64.zip $PROJECT.64.exe
cd ..
rm -rf tmp
