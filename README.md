# minicontainer

Minimal container engine built in Go from scratch, using Linux namespaces, cgroups, and chroot.

## Features

- PID, UTS, mount, network namespaces
- Filesystem isolation via `chroot`
- Memory and CPU limits with cgroups v2
- Runs ubuntu-based rootfs

## Prerequisites

- Linux (Ubuntu 20.04+ or similar)
- Go 1.19+
- Root privileges (`sudo`)

## Build & Run

```bash
go build -o minicontainer .
sudo ./minicontainer run /bin/sh
```
