---
title: "Installation"
description: "Install wiki with Go, Homebrew, Scoop, a Linux package, Docker, or a prebuilt binary."
weight: 2
---

wiki ships as a single static binary for Linux, macOS, Windows, and FreeBSD on
amd64 and arm64. Pick whichever method fits your setup.

## Go

```bash
go install github.com/tamnd/wikipedia-cli/cmd/wiki@latest
```

The binary lands in `$(go env GOPATH)/bin`. Requires Go 1.26 or newer.

## Homebrew

Once the tap is published:

```bash
brew install tamnd/tap/wiki
```

## Scoop (Windows)

```powershell
scoop bucket add tamnd https://github.com/tamnd/scoop-bucket
scoop install wiki
```

## Linux packages

Download the `.deb`, `.rpm`, or `.apk` for your architecture from the
[releases page](https://github.com/tamnd/wikipedia-cli/releases) and install it:

```bash
sudo dpkg -i wiki_*_linux_amd64.deb     # Debian / Ubuntu
sudo rpm -i  wiki_*_linux_amd64.rpm     # Fedora / RHEL
sudo apk add --allow-untrusted wiki_*_linux_amd64.apk   # Alpine
```

## Docker

```bash
docker run --rm ghcr.io/tamnd/wiki read "Alan Turing"
```

Mount a volume to keep the cache and downloads between runs:

```bash
docker run --rm -v ~/data/wiki:/data ghcr.io/tamnd/wiki \
  dump list --wiki simplewiki
```

## Prebuilt binary

Grab the archive for your platform from the
[releases page](https://github.com/tamnd/wikipedia-cli/releases), extract it,
and put `wiki` on your `PATH`:

```bash
tar xzf wiki_*_linux_amd64.tar.gz
sudo mv wiki /usr/local/bin/
```

Every release also carries `checksums.txt`, a cosign signature, and a
CycloneDX SBOM per archive if you want to verify what you downloaded.

## Verify

```bash
wiki version
wiki read "Wikipedia" --summary
```

Next: [the quick start](/getting-started/quick-start/).
