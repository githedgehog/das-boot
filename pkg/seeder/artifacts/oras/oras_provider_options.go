// Copyright 2023 Hedgehog
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
