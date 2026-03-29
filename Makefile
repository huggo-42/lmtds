.PHONY: build run build-and-run

build:
	go build

run:
	./lmtds generate -m migrations -o dbdiagram.dbml

build-and-run: build run
