#!/bin/bash

# Terminate any existing k13d processes
killall k13d


# Clean previous build artifacts
rm -rf ./build/*

# Build the application
make build

# Run the application with web mode and local authentication
./build/k13d --cli
