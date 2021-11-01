#!/bin/bash

BUILDDATE=`date +%Y%m%d`
COMMIT=`git rev-parse --short HEAD`
VERSION=`git describe --tags --exclude latest`

EXIT_STATUS=0

LDFLAGS="-X version.Program=adalanche -X version.Builddate=$BUILDDATE -X version.Commit=$COMMIT -X version.Version=$VERSION"

GOOS=windows GOARCH=amd64 go build -ldflags "$LDFLAGS" -o adalanche-windows-x64-$VERSION.exe ./adalanche || EXIT_STATUS=$?
GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o adalanche-osx-x64-$VERSION ./adalanche || EXIT_STATUS=$?
GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o adalanche-linux-x64-$VERSION ./adalanche || EXIT_STATUS=$?

LDFLAGS="-X version.Program=adalanche-collector -X version.Builddate=$BUILDDATE -X version.Commit=$COMMIT -X version.Version=$VERSION"

GOOS=windows GOARCH=386 go build -ldflags "$LDFLAGS" -o adalanche-collector-windows-386-$VERSION.exe ./collector || EXIT_STATUS=$?

exit $EXIT_STATUS
