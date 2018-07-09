#!/usr/bin/env bash
set -e
cd src

echo "Current dir: $PWD"
ls -l

echo "Cloning cube executor from git..."
git clone https://github.com/akaumov/cube_executor.git cube_executor

echo "Current dir: $PWD"
ls -l

cd cube_executor
sed -i "s|github.com/akaumov/cube-stub|$CUBE_PACKAGE|g" cube.go

echo "Current dir: $PWD"
ls -l
echo "Cloning cube handler from git..."
dep ensure

echo "Current dir: $PWD"
ls -l
echo "Compiling code..."
go build -x -v  ./cmd/cube

echo "Current dir: $PWD"
chmod u=rx,g=rx,o=rx cube
ls -l
echo "Making tar..."
tar -cvf cube.tar cube

echo "Current dir: $PWD"
ls -l
echo "Moving..."
mv cube.tar /build/cube.tar

chmod u=rw,g=rw,o=rw /build/cube.tar

echo "Moved"
cd /build
echo "Current dir: $PWD"
ls -l