#!/bin/bash
export LD_LIBRARY_PATH=$(pwd)/clib:$LD_LIBRARY_PATH
go run go/cmd/api/main.go