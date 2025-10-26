#!/bin/bash

cd "$(dirname "$0")"

echo "=== Проверка библиотеки ==="
if [ ! -f "clib/build/libengine.so" ]; then
    echo "Библиотека не найдена! Собираем..."
    cd clib
    mkdir -p build
    cd build
    cmake ..
    make
    cd ../..
fi

echo "=== Запуск тестов ==="
export LD_LIBRARY_PATH=$(pwd)/clib/build:$LD_LIBRARY_PATH
export CGO_LDFLAGS="-L$(pwd)/clib/build -lengine -lcrypto"
export CGO_CFLAGS="-I$(pwd)/clib"

echo "LD_LIBRARY_PATH: $LD_LIBRARY_PATH"
echo "CGO_LDFLAGS: $CGO_LDFLAGS"

go test ./go/internal/cgobridge 