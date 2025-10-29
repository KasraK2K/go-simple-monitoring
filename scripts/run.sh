#!/bin/bash

# Run the monitoring backend without C compiler warnings
export CGO_CFLAGS="-Wno-gnu-folding-constant"
go run ./cmd