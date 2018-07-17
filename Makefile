VERSION=$(shell git describe --tags --abbrev=0)

nss_docker.so: *.go
	go build -o nss_docker.so -buildmode=c-shared -ldflags '-extldflags "-Wl,-soname,nss_docker.so.2"'

# dirty hack to avoid needing autoconf
# would be nice to also get SONAME dynamically
install: nss_docker.so
	$(eval TARGET := $(shell dirname $(shell ldd nss_docker.so | grep libc.so | awk '{ print $$3 }')))
	install -D nss_docker.so $(DESTDIR)$(TARGET)/nss_docker-$(VERSION).so
