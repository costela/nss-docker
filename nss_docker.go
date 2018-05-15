/*
Copyright © 2018 Leo Antunes <leo@costela.net>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

// #include <stdlib.h>
// #include <errno.h>
// #include <nss.h>
// #include <netdb.h>
// #include <arpa/inet.h>
import "C"
import (
	"context"
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strings"
	"unsafe"

	"docker.io/go-docker"
	"docker.io/go-docker/api/types"
)

func init() {
	runtime.GOMAXPROCS(1) // we don't need extra goroutines

	cfgFiles := []string{"/etc/nss_docker.json"}
	if usr, err := user.Current(); err == nil {
		cfgFiles = append([]string{fmt.Sprintf("%s/.nss_docker.json", usr.HomeDir)}, cfgFiles...)
	}

	// TODO: what should we do on parse-errors? log.Fatal's seems a bit overkill
	for _, file := range cfgFiles {
		if configFile, err := os.Open(file); err == nil {
			defer configFile.Close()
			_ = parseConfig(configFile)
			return
		}
	}
}

//export _nss_docker_gethostbyname3_r
func _nss_docker_gethostbyname3_r(name *C.char, af C.int, result *C.struct_hostent,
	buffer *C.char, buflen C.size_t, errnop *C.int, herrnop *C.int, ttlp *C.int32_t,
	canonp **C.char) C.enum_nss_status {

	if af == C.AF_UNSPEC {
		af = C.AF_INET
	}

	if af != C.AF_INET {
		return unavailable(errnop, herrnop)
	}

	queryName := C.GoString(name)

	if len(queryName) == 0 || !strings.HasSuffix(queryName, config.Suffix) {
		return unavailable(errnop, herrnop)
	}

	client, err := docker.NewEnvClient()
	if err != nil {
		return unavailable(errnop, herrnop)
	}
	defer client.Close()

	_, addresses, err := queryDockerForName(client, queryName)
	if err != nil {
		return unavailable(errnop, herrnop)
	}

	if len(addresses) == 0 {
		return notfound(errnop, herrnop)
	}

	// buffer must fit addresses and respective pointers + 1 (NULL pointer)
	cAddressesSize := C.size_t(len(addresses)) * C.sizeof_struct_in_addr
	cAddressPtrsSize := uintptr(len(addresses)+1) * unsafe.Sizeof(uintptr(0))
	if buflen < (cAddressesSize + C.size_t(cAddressPtrsSize)) {
		return bufferTooSmall(errnop, herrnop)
	}

	// TODO: is there really no cleaner way to access the data as an array?
	cAddressPtrs := (*[1 << 30]*C.char)(unsafe.Pointer(buffer))
	cAddresses := (*[1 << 30]*C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(buffer)) + cAddressPtrsSize))
	for i, a := range addresses {
		cAddressPtrs[i] = (*C.char)(unsafe.Pointer(&cAddresses[i]))
		if ret := C.inet_aton(C.CString(a), (*C.struct_in_addr)(unsafe.Pointer(&cAddresses[i]))); ret != C.int(1) {
			return unavailable(errnop, herrnop)
		}
	}
	cAddressPtrs[len(addresses)] = nil

	result.h_name = name
	result.h_aliases = (**C.char)(unsafe.Pointer(&cAddressPtrs[len(addresses)])) // TODO: actually build alias-list
	result.h_addrtype = af
	result.h_length = C.sizeof_struct_in_addr
	result.h_addr_list = (**C.char)(unsafe.Pointer(buffer))

	return C.NSS_STATUS_SUCCESS
}

//export _nss_docker_gethostbyname2_r
func _nss_docker_gethostbyname2_r(name *C.char, af C.int, result *C.struct_hostent,
	buffer *C.char, buflen C.size_t, errnop *C.int, herrnop *C.int) C.enum_nss_status {
	return _nss_docker_gethostbyname3_r(name, af, result, buffer, buflen, errnop, herrnop, nil, nil)
}

//export _nss_docker_gethostbyname_r
func _nss_docker_gethostbyname_r(name *C.char, result *C.struct_hostent, buffer *C.char,
	buflen C.size_t, errnop *C.int, herrnop *C.int) C.enum_nss_status {
	return _nss_docker_gethostbyname3_r(name, C.AF_UNSPEC, result, buffer, buflen, errnop, herrnop, nil, nil)
}

func unavailable(errnop, herrnop *C.int) C.enum_nss_status {
	*errnop = C.ENOENT
	*herrnop = C.NO_DATA
	return C.NSS_STATUS_UNAVAIL
}

func retry(errnop, herrnop *C.int) C.enum_nss_status {
	*errnop = C.EAGAIN
	*herrnop = C.NO_RECOVERY
	return C.NSS_STATUS_TRYAGAIN
}

func bufferTooSmall(errnop, herrnop *C.int) C.enum_nss_status {
	*errnop = C.ERANGE
	*herrnop = C.NETDB_INTERNAL
	return C.NSS_STATUS_TRYAGAIN
}

func notfound(errnop *C.int, herrnop *C.int) C.enum_nss_status {
	*errnop = C.ENOENT
	*herrnop = C.HOST_NOT_FOUND
	return C.NSS_STATUS_NOTFOUND
}

func queryDockerForName(client *docker.Client, fqdn string) (aliases, addresses []string, err error) {
	containers, err := client.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return nil, nil, err
	}

	hostname := strings.TrimSuffix(fqdn, config.Suffix)

	var tmpAliases []string
	var tmpAddresses []string
	var found bool
	for _, container := range containers {
		found = false
		tmpAliases = []string{}
		tmpAddresses = []string{}

		maybeAppendProject := func(s string, doit bool) string {
			if p, ok := container.Labels["com.docker.compose.project"]; ok && doit {
				return fmt.Sprintf("%s.%s", s, p)
			}
			return s
		}

		// ContainerList does not return all info, like Aliases
		// see: curl --unix-socket /var/run/docker.sock http://localhost/containers/json
		containerJSON, err := client.ContainerInspect(context.Background(), container.ID)
		if err != nil {
			return nil, nil, err
		}

		// names are trimmed for the compose case, but more useful for the non-compose case
		for _, name := range container.Names {
			name = maybeAppendProject(strings.Trim(name, "/"), config.IncludeComposeProject)
			found = (found || name == hostname)
			tmpAliases = append(tmpAliases, fmt.Sprintf("%s%s", name, config.Suffix))
		}

		for _, endpoint := range containerJSON.NetworkSettings.Networks {
			tmpAddresses = append(tmpAddresses, endpoint.IPAddress)
			for _, alias := range endpoint.Aliases {
				alias = maybeAppendProject(alias, config.IncludeComposeProject)
				found = (found || alias == hostname)
				tmpAliases = append(tmpAliases, fmt.Sprintf("%s%s", alias, config.Suffix))
			}
		}

		if found {
			aliases = append(aliases, tmpAliases...)
			addresses = append(addresses, tmpAddresses...)
		}
	}
	return aliases, addresses, nil
}

func main() {}