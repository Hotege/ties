#!/bin/bash

export LD_LIBRARY_PATH=`pwd`/libs:$LD_LIBRARY_PATH

./outputs/ties -libs `pwd`/libs $*
