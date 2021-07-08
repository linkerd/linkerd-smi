#!/usr/bin/env bash

set -eu

os() {
  os=$(uname -s)
  case $os in
    CYGWIN* | MINGW64*)
      echo "windows"
      ;;
    Darwin)
      echo "darwin"
      ;;
    Linux)
      echo "linux"
      ;;
    *)
      echo "unsupported os: $os" >&2
      exit 1
      ;;
  esac
}

arch() {
  arch=$(uname -m)
  case $arch in
    x86_64)
      echo "amd64"
      ;;
    armv8*)
      echo "arm64"
      ;;
    aarch64*)
      echo "arm64"
      ;;
    armv*)
      echo "arm"
      ;;
    amd64|arm64)
      echo "$arch"
      ;;
    *)
      echo "unsupported architecture: $arch" >&2
      exit 1
      ;;
      esac
}
