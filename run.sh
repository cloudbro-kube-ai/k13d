#!/bin/bash

# Build the application
make build

# Run the application with web mode and local authentication
./build/k13d -web -port 8081 --auth-mode local --admin-user admin --admin-password secret
