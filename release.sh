#!/bin/sh
PROJECT='ns2query'
export CGO_ENABLED=0
rm -rf release
mkdir release
mkdir tmp
GOARCH=mipsle go build -ldflags='-s -w -extldflags -static' -o tmp/$PROJECT.mipsel
7z a release/$PROJECT.mipsel.zip tmp/$PROJECT.mipsel
GOARCH=arm go build -ldflags='-s -w -extldflags -static' -o tmp/$PROJECT.arm
7z a release/$PROJECT.arm.zip tmp/$PROJECT.arm
GOARCH=386 go build -ldflags='-s -w -extldflags -static' -o tmp/$PROJECT.386
7z a release/$PROJECT.linux32.zip tmp/$PROJECT.386
GOARCH=amd64 go build -ldflags='-s -w -extldflags -static' -o tmp/$PROJECT.amd64
7z a release/$PROJECT.linux64.zip tmp/$PROJECT.amd64
GOOS=windows GOARCH=386 go build -ldflags='-s -w -extldflags -static' -o tmp/$PROJECT.32.exe
7z a release/$PROJECT.win32.zip tmp/$PROJECT.32.exe
GOOS=windows GOARCH=amd64 go build -ldflags='-s -w -extldflags -static' -o tmp/$PROJECT.64.exe
7z a release/$PROJECT.win64.zip tmp/$PROJECT.64.exe
rm -rf tmp
