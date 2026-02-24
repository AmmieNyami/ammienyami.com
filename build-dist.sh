#!/bin/sh

if ! command -v musl-gcc > /dev/null 2>&1; then
    echo "ERROR: command \`musl-gcc\` is required build a distribution" 1>&2
    exit 1
fi

cd backend
CC=musl-gcc go build -ldflags '-linkmode external -extldflags "-static"'
cd ..

rm -rf build
mkdir -p build
cp -rL ./{pages,static,templates} ./build/
cp backend/backend ./build/backend

cd build
tar -czf ../dist.tar.gz *
