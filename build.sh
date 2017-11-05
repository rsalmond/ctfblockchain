#!/bin/bash

echo "building linux"
go build
scp ctfblockchain box:~/

echo "building osx"
GOOS=darwin GOARCH=amd64 go build -o ctfblockchain-osx
scp ctfblockchain-osx box:~/

echo "building windows"
GOOS=windows GOARCH=amd64 go build -o ctfblockchain-win64.exe
scp ctfblockchain-win64.exe box:~/
