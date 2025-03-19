#!/bin/sh
if [ -n "gitea" ]; then mv gitea gitea-tmp; fi
TAGS="bindata timetzdata sqlite sqlite_unlock_notify" GOOS="linux" GOARCH="amd64" make build
docker buildx build --platform linux/amd64 -f Dockerfile.manual2 -t gitea .
if [ -n "gitea-tmp" ]; then mv gitea-tmp gitea; fi
