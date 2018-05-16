all:
	go build -o nss_docker.so -buildmode=c-shared *.go

# dirty hack to avoid needing autoconf
# would be nice to also get SONAME dynamically
install: all
	install nss_docker.so $(shell dirname $(shell readlink -f $(shell whereis -b libnss_dns | awk '{ print $$2 }')))/libnss_docker.so.2

