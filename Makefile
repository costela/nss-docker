all:
	go build -o nss_docker.so -buildmode=c-shared *.go

install: all
	install nss_docker.so $(shell dirname $(shell whereis -b libc | awk '{ print $$2 }'))