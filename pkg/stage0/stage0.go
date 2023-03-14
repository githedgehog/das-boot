package stage0

import (
	"crypto/x509"
	"fmt"
	"io"
	"os"

	"go.githedgehog.com/dasboot/pkg/config"
	configstage "go.githedgehog.com/dasboot/pkg/stage0/config"
)

func ReadConfig() (*configstage.Stage0, error) {
	// open and read executable into memory
	exePath, err := os.Executable()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(exePath)
	if err != nil {
		return nil, err
	}

	exeBytes, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	// now read embedded config for the first time - ignoring signature verification this time
	// an embedded config must exist, so even without signature validation, if this steps fail, we fail
	var cfg configstage.Stage0
	if err := config.ReadEmbeddedConfig(exeBytes, &cfg, nil, config.ReadOptionIgnoreSignature); err != nil {
		return nil, err
	}

	// perform signature validation if a CA is embedded at this point
	if len(cfg.SignatureCA) > 0 {
		// NOTE: this is a chicken & egg problem: using an embedded CA cert here
		// does not add any real security - with the exception of making things harder
		// for attackers :)
		// The Signature CA certificate should ideally come from the attached USB stick (location partition).
		// However, as the USB stick is most likely "downloadable" from the factory, I'm not sure we'll be
		// able to deal with this that way. Anyway, so this prepares things in the right way.
		//
		// Parse Signature CA cert and create a cert pool for it
		signatureCACert, err := x509.ParseCertificate(cfg.SignatureCA)
		if err != nil {
			return nil, err
		}
		configSignatureCAPool := x509.NewCertPool()
		configSignatureCAPool.AddCert(signatureCACert)

		// now read embedded configuration again, but this time only ignore wrong times at certificate validation time
		// NOTE: this is okay, as we haven't run NTP yet, and the time might not be up-to-date
		if err := config.ReadEmbeddedConfig(exeBytes, &cfg, configSignatureCAPool, config.ReadOptionIgnoreExpiryTime); err != nil {
			return nil, err
		}
	}

	// this completes reading the stage0 configuration
	return &cfg, nil
}

func Run(override *configstage.Stage0) error {
	// read the embedded configuration first
	embedded, err := ReadConfig()
	if err != nil {
		return fmt.Errorf("stage0: reading embedded config: %w", err)
	}

	// Merge configs with override
	cfg := configstage.MergeConfigs(embedded, override)
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("stage0: config validation: %w", err)
	}

	// now let's start doing stuff

	return nil
}
