#!/bin/bash

g++ cxx/addons.cpp -shared -fPIC -s -o libs/libaddons.so \
-I addons \
-lboost_filesystem

g++ subjects/hello/hello.cpp -shared -fPIC -s -o libs/hello.tie \
-I addons

go build -ldflags="-s -w" -o outputs/ties
