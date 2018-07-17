nss_docker.so: *.go
	go build -o nss_docker.so -buildmode=c-shared -ldflags '-extldflags "-Wl,-soname,nss_docker.so.2"'

# dirty hack to avoid needing autoconf
# would be nice to also get SONAME dynamically
install: nss_docker.so
	install nss_docker.so $(shell dirname $(shell readlink -f $(shell whereis -b libnss_dns | awk '{ print $$2 }')))/libnss_docker.so.2

