package oras

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"time"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/seeder/artifacts"
	"go.uber.org/zap"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

type orasProvider struct {
	ctx context.Context

	serverCAPath   string
	clientCertPath string
	clientKeyPath  string
	username       string
	password       string
	accessToken    string
	refreshToken   string

	url      *url.URL
	registry *remote.Registry
}

var _ artifacts.Provider = &orasProvider{}

func Provider(ctx context.Context, registryURL string, options ...ProviderOption) (artifacts.Provider, error) {
	var err error
	// apply options
	ret := &orasProvider{
		ctx: ctx,
	}
	for _, opt := range options {
		opt(ret)
	}

	// parse URL
	ret.url, err = url.Parse(registryURL)
	if err != nil {
		return nil, fmt.Errorf("parsing registry URL: %w", err)
	}
	if ret.url.Scheme != "oci" {
		return nil, fmt.Errorf("registry URL must have OCI scheme, got '%s'", ret.url.Scheme)
	}

	ret.registry, err = remote.NewRegistry(ret.url.Host)
	if err != nil {
		return nil, fmt.Errorf("create ORAS client: %w", err)
	}

	creds := func(_ context.Context, target string) (auth.Credential, error) {
		if ret.username != "" || ret.password != "" || ret.accessToken != "" || ret.refreshToken != "" {
			if target == ret.url.Host {
				return auth.Credential{
					Username:     ret.username,
					Password:     ret.password,
					AccessToken:  ret.accessToken,
					RefreshToken: ret.refreshToken,
				}, nil
			}
		}
		return auth.EmptyCredential, nil
	}

	ret.registry.Client = &auth.Client{
		Credential: creds,
		Cache:      auth.NewCache(),
		Client: &http.Client{
			Transport: &http.Transport{
				// take proxy from the environment if set
				Proxy: http.ProxyFromEnvironment,

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
				MaxConnsPerHost:   3,
				IdleConnTimeout:   90 * time.Second,

				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,

				// as we are setting our own DialContext and TLSClientConfig
				// Go internally disables trying to use HTTP/2 (why?)
				// so we are reenabling this here
				ForceAttemptHTTP2: true,

				// Our TLS configuration that we prepped before
				TLSClientConfig: &tls.Config{
					Rand:         rand.Reader,
					Time:         time.Now,
					RootCAs:      caPool(ret.serverCAPath),
					Certificates: clientCertificates(ret.clientCertPath, ret.clientKeyPath),
					MinVersion:   tls.VersionTLS12,
				},
			},
		},
	}

	return nil, nil
}

// Get implements artifacts.Provider
func (op *orasProvider) Get(artifact string) io.ReadCloser {
	ctx, cancel := context.WithTimeout(op.ctx, time.Second*60)
	defer cancel()

	// build repo name from artifact
	repoName := path.Join(op.url.Path, artifact)
	src, err := op.registry.Repository(ctx, repoName)
	if err != nil {
		log.L().Error("oras: getting repository reference failed", zap.String("repo", repoName), zap.Error(err))
	}

	// TODO: tag name
	tagName := "latest"

	// downloads the stuff locally
	dst := memory.New()
	desc, err := oras.Copy(ctx, src, tagName, dst, tagName, oras.DefaultCopyOptions)
	if err != nil {
		log.L().Error("oras: copying artifact into memory failed", zap.Error(err))
	}

	// this actually fetches the layer
	rc, err := dst.Fetch(ctx, desc)
	if err != nil {
		log.L().Error("oras: fetch layer content failed", zap.Error(err))
	}

	// and it has the perfect return type - nice!
	return rc
}
