#!/usr/bin/env bash
mkdir src
cd src

echo "Cloning cube executor from git..."
git clone https://github.com/akaumov/cube_executor.git cube_executor

cd cube_executor
sed -i "s|cube_executor/test_cube|$CUBE_PACKAGE|g" cube.go

echo "Cloning cube handler from git..."
dep ensure

echo "Compiling code..."
go build -x -v -o ./build/cube ./cmd/cube