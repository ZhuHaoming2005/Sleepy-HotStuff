#!/bin/bash

# A script to build all executables offline,
# and prepare them for running.

# Exit immediately if a command exits with a non-zero status.
set -e

# --- Environment Setup ---

# Enable Go Modules.
# This ensures the build process uses the go.mod file for dependency management.
go env -w GO111MODULE=on

echo "Starting build process using vendored dependencies ..."

# Build the executables using the -mod=vendor flag.
# This forces the Go compiler to use the local 'vendor' directory
# and prevents any network access for dependencies.

go build -mod=vendor -o ./ecdsagen ./src/main/ecdsagen
chmod +x ./ecdsagen
./ecdsagen 0 100

go build -mod=vendor -o ./server ./src/main/server
chmod +x ./server

go build -mod=vendor -o ./client ./src/main/client
chmod +x ./client

echo "Build finished successfully!"

# List the generated binaries to confirm they were created.
ls -l ecdsagen server client
