package oras

type ProviderOption func(*orasProvider)

func ProviderOptionServerCA(path string) func(*orasProvider) {
	return func(op *orasProvider) {
		op.serverCAPath = path
	}
}

func ProviderOptionTLSClientAuth(certPath, keyPath string) func(*orasProvider) {
	return func(op *orasProvider) {
		op.clientCertPath = certPath
		op.clientKeyPath = keyPath
	}
}

func ProviderOptionBasicAuth(username, password string) func(*orasProvider) {
	return func(op *orasProvider) {
		op.username = username
		op.password = password
	}
}

func ProviderOptionAccessToken(accessToken string) func(*orasProvider) {
	return func(op *orasProvider) {
		op.accessToken = accessToken
	}
}

func ProviderOptionRefreshToken(refreshToken string) func(*orasProvider) {
	return func(op *orasProvider) {
		op.refreshToken = refreshToken
	}
}
