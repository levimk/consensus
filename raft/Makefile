.DEFAULT_GOAL := build
# current_dir := $(notdir $(patsubst %/,%,$(dir $(mkfile_path))))
current_dir = $(shell pwd)
ID?=1
SERVER?=false
fmt:
	go fmt ./...
.PHONY:fmt

lint: fmt
	golint ./...
.PHONY:lint

vet: fmt
	go vet ./...
.PHONY:vet

build: vet
	go build .
.PHONY:build

tidy: vet
	go mod tidy
.PHONY: tidy

run:
	go run . -serveClient=$(SERVER) -id=$(ID) -config=$(current_dir)/consensus/config
.PHONY:run

runCluster:
	go run . -serveClient=true -id=1 -config=$(current_dir)/consensus/config
	go run . -serveClient=true -id=2 -config=$(current_dir)/consensus/config
	go run . -serveClient=true -id=3 -config=$(current_dir)/consensus/config
	go run . -serveClient=true -id=4 -config=$(current_dir)/consensus/config
	go run . -serveClient=true -id=5 -config=$(current_dir)/consensus/config
.PHONY:runCluster

# What is ".PHONY"?
# https://stackoverflow.com/questions/2145590/what-is-the-purpose-of-phony-in-a-makefile