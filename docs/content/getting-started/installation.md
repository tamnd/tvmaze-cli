---
title: "Installation"
description: "Install tvmaze from a release, with go install, or from source."
weight: 20
---

## Prebuilt binaries

Every [release](https://github.com/tamnd/tvmaze-cli/releases) carries archives for Linux, macOS,
and Windows on amd64 and arm64, plus deb, rpm, and apk packages for Linux.
Download, unpack, put `tvmaze` on your `PATH`, done. The `checksums.txt`
on each release is signed with keyless [cosign](https://docs.sigstore.dev/) if
you want to verify before running.

## With Go

```bash
go install github.com/tamnd/tvmaze-cli/cmd/tvmaze@latest
```

That puts `tvmaze` in `$(go env GOPATH)/bin`, which is `~/go/bin` unless
you moved it. Make sure that directory is on your `PATH`.

## From source

```bash
git clone https://github.com/tamnd/tvmaze-cli
cd tvmaze-cli
make build        # produces ./bin/tvmaze
./bin/tvmaze version
```

## Container image

```bash
docker run --rm ghcr.io/tamnd/tvmaze:latest --help
```

## Checking the install

```bash
tvmaze version
```

prints the version and exits.
