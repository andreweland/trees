#!/bin/sh
export GOPATH=${PWD}:${GOPATH}
go install trees/frontend
