package seeder

import "net/http"

type server struct {
	httpServers []*httpServer
}

func newServer(b *BindInfo, handler http.Handler) (*server, error) {
	if len(b.Address) == 0 {
		return nil, invalidConfigError("no address in server config")
	}
	if (b.ServerKeyPath != "" && b.ServerCertPath == "") || (b.ServerCertPath != "" && b.ServerKeyPath == "") {
		return nil, invalidConfigError("server key and server cert must always be set together")
	}

	ret := &server{}
	for _, addr := range b.Address {
		if addr == "" {
			return nil, invalidConfigError("address must not be empty")
		}
		ret.httpServers = append(ret.httpServers, newHttpServer(addr, b.ServerKeyPath, b.ServerCertPath, b.ClientCAPath, handler))
	}
	return ret, nil
}
