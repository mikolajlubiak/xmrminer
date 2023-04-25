#!/usr/bin/env sh
GOOS=windows GOARCH=amd64 garble build --ldflags "-H=windowsgui"
