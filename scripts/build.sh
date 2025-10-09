#!/bin/bash

# A script to set up the Go environment, build all executables,
# and prepare them for running.

# Exit immediately if a command exits with a non-zero status.
set -e

# --- Environment Setup ---

echo "INFO: Setting Go environment variables..."
# This command only needs to be run once, but it's safe to run multiple times.
go env -w GO111MODULE=on

echo "INFO: Downloading Go module dependencies..."
# This will be silent if dependencies are already downloaded.
go mod download


# --- Build Executables ---

# Note: We are building from the project root directory.
# The paths to the main packages are specified relative to the root.
# The output (-o) files will be placed in the current (root) directory.

echo "INFO: Building 'ecdsagen' executable..."
go build -o ./ecdsagen ./src/main/ecdsagen/
chmod +x ./ecdsagen
echo "SUCCESS: 'ecdsagen' built and made executable."


# We can run it here or the user can run it manually later.
echo "INFO: Running ecdsagen to generate keys..."
./ecdsagen 0 100

echo "INFO: Building 'server' executable..."
go build -o ./server ./src/main/server/
chmod +x ./server
echo "SUCCESS: 'server' built and made executable."

echo "INFO: Building 'client' executable..."
go build -o ./client ./src/main/client/
chmod +x ./client
echo "SUCCESS: 'client' built and made executable."


echo ""
echo "-------------------------------------"
echo "ALL BUILDS COMPLETED SUCCESSFULLY!"
echo "Executables (ecdsagen, server, client) are now in the project root directory."
echo "-------------------------------------"
