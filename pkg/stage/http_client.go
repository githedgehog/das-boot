package stage

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"time"

	"go.githedgehog.com/dasboot/pkg/partitions/identity"
)

type HTTPClientOption int

const (
	// HTTPClientOptionUndefined should not be used
	HTTPClientOptionUndefined HTTPClientOption = iota

	// HTTPClientOptionServerCertificateIgnoreExpiryTime can be used to request to ignore
	// certificate validation errors during the TLS handshake which are related to the
	// fact that the server certificate is expired. This is particularly helpful in
	// circumstances when we cannot trust our own system clock which is the case before
	// we have applied time from NTP servers.
	HTTPClientOptionServerCertificateIgnoreExpiryTime
)

// SeederHTTPClient will create an HTTP client which can be used in interaction with the seeder
func SeederHTTPClient(serverCA []byte, ip identity.IdentityPartition, options ...HTTPClientOption) (*http.Client, error) {
	// server CA
	serverCACert, err := x509.ParseCertificate(serverCA)
	if err != nil {
		return nil, err
	}
	serverCAPool := x509.NewCertPool()
	serverCAPool.AddCert(serverCACert)

	// build client certificates
	clientCertificates := []tls.Certificate{}
	if ip != nil && ip.HasClientKey() && ip.HasClientCert() {
		clientCert, err := ip.LoadX509KeyPair()
		if err != nil {
			return nil, err
		}
		clientCertificates = append(clientCertificates, clientCert)
	}

	// rand could get swapped out for the TPM rand
	rand := rand.Reader

	// process options
	var ignoreExpiry bool
	for _, option := range options {
		if option == HTTPClientOptionServerCertificateIgnoreExpiryTime {
			ignoreExpiry = true
			break
		}
	}

	timeFunc := time.Now
	if ignoreExpiry { //nolint:staticcheck
		// TODO: we haven't seen a server certificate yet at this point
		// so how to accomodate this
	}

	return &http.Client{
		// TODO: think about this: we are serving large artifacts
		// no need to limit us here, all connection internals
		// are handled in more detail below anyways
		// Timeout: time.Second * 90,

		Transport: &http.Transport{
			// disable any proxies specifically here
			Proxy: nil,

			// There are no connection timeouts
			// so we are doing pretty much exactly what
			// Go is doing itself
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				// increasing this from the default Go settings
				// as we can ensure that if there is IPv6 in our network
				// it actually *must* be configured correctly.
				FallbackDelay: 600 * time.Millisecond,
			}).DialContext,

			// These are HTTP keep alives (not TCP keepalives)
			// and their corresponding idle connection settings and timeouts
			DisableKeepAlives: false,
			MaxIdleConns:      10,
			MaxConnsPerHost:   1,
			IdleConnTimeout:   90 * time.Second,

			TLSHandshakeTimeout: 10 * time.Second,
			// TODO: think about this: we are serving large artifacts
			// which need to be prepared server side before any response
			// header can be written
			// ResponseHeaderTimeout: 15 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,

			// as we are setting our own DialContext and TLSClientConfig
			// Go internally disables trying to use HTTP/2 (why?)
			// so we are reenabling this here
			ForceAttemptHTTP2: true,

			// Our TLS configuration that we prepped before
			TLSClientConfig: &tls.Config{
				Rand:         rand,
				Time:         timeFunc,
				RootCAs:      serverCAPool,
				Certificates: clientCertificates,
				MinVersion:   tls.VersionTLS12,
			},
		},
	}, nil
}
