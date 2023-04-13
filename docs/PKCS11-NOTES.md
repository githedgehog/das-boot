# cosign pkcs11-tool list-tokens --module-path ~/lib/yubihsm_pkcs11.so

Listing tokens of PKCS11 module '/home/mheese/lib/yubihsm_pkcs11.so'
Token in slot 0
        Label: YubiHSM
        Manufacturer: Yubico (www.yubico.com)
        Model: YubiHSM
        S/N: 18952614

 
# cosign pkcs11-tool list-keys-uris --module-path ~/lib/yubihsm_pkcs11.so
Enter PIN for PKCS11 token 'YubiHSM': 
Listing URIs of keys in slot '0' of PKCS11 module '/home/mheese/lib/yubihsm_pkcs11.so'
Object 0
        Label: label_ecdsa_sign
        ID: 0064
        URI: pkcs11:token=YubiHSM;slot-id=0;id=%00%64;object=label_ecdsa_sign?module-path=/home/mheese/lib/yubihsm_pkcs11.so&pin-value=0001password

# cosign sign --key 'pkcs11:token=YubiHSM;slot-id=0;id=%00%64;object=label_ecdsa_sign?module-path=/home/mheese/lib/yubihsm_pkcs11.so&pin-value=0001password' registry.local:5000/githedgehog/das-boot
WARNING: no x509 certificate retrieved from the PKCS11 token
WARNING: Image reference registry.local:5000/githedgehog/das-boot uses a tag, not a digest, to identify the image to sign.
    This can lead you to sign a different image than the intended one. Please use a
    digest (example.com/ubuntu@sha256:abc123...) rather than tag
    (example.com/ubuntu:latest) for the input to cosign. The ability to refer to
    images by tag will be removed in a future release.


        Note that there may be personally identifiable information associated with this signed artifact.
        This may include the email address associated with the account with which you authenticate.
        This information will be used for signing this artifact and will be stored in public transparency logs and cannot be removed later.

By typing 'y', you attest that you grant (or have permission to grant) and agree to have this information stored permanently in transparency logs.
Are you sure you would like to continue? [y/N] y
tlog entry created with index: 17126562
Pushing signature to: registry.local:5000/githedgehog/das-boot

# cosign verify --key 'pkcs11:token=YubiHSM;slot-id=0;id=%00%64;object=label_ecdsa_sign?module-path=/home/mheese/lib/yubihsm_pkcs11.so&pin-value=0001password' registry.local:5000/githedgehog/das-boot

Verification for registry.local:5000/githedgehog/das-boot:latest --
The following checks were performed on each of these signatures:
  - The cosign claims were validated
  - Existence of the claims in the transparency log was verified offline
  - The signatures were verified against the specified public key

[{"critical":{"identity":{"docker-reference":"registry.local:5000/githedgehog/das-boot"},"image":{"docker-manifest-digest":"sha256:198a599d5830352efb06b988ab23cddcf9c93bc9824ae88ab24d6ab425330b29"},"type":"cosign container image signature"},"optional":{"Bundle":{"SignedEntryTimestamp":"MEYCIQC+l0bVpu2xW/0/QIvRx0bIrXuKPm67hGeDlm5LIV0w3wIhALDDFi9vkZuAEzSPB6XNTtH7BQCPsrSllfDp9IPeL5ow","Payload":{"body":"eyJhcGlWZXJzaW9uIjoiMC4wLjEiLCJraW5kIjoiaGFzaGVkcmVrb3JkIiwic3BlYyI6eyJkYXRhIjp7Imhhc2giOnsiYWxnb3JpdGhtIjoic2hhMjU2IiwidmFsdWUiOiJiNjJhZTIxNGYyYjgxMDA1NTIyNzU5Yzc4M2U3MzRmZjRmMzU0NzU5MmE2ZTUzZDMwOTVhZDM4NmQwNjc5ZWM5In19LCJzaWduYXR1cmUiOnsiY29udGVudCI6Ik1FWUNJUUNYVGQ4alRUcDFKN3h3aXRSdENldzgyMnErVERWZVF0WkZVNFgzYnVzc0NBSWhBUE0zNmQ5blhRV2QrQml3d0VVTW9EY0YvNDk0dnpZTVovaTZJaGk3d2k3QiIsInB1YmxpY0tleSI6eyJjb250ZW50IjoiTFMwdExTMUNSVWRKVGlCUVZVSk1TVU1nUzBWWkxTMHRMUzBLVFVacmQwVjNXVWhMYjFwSmVtb3dRMEZSV1VsTGIxcEplbW93UkVGUlkwUlJaMEZGSzNWNmJpOU9jbUl6TjFGbGNqZFNjalo0Y20xb1lYbEpNakZ2VGdweFowSm1hREo0YlRseFNVaENWakpDUW1STGNrSnZWelk1YVVJMmF6aEJkMkpQTlc5TmNWZGxiV0pvWmpkTlZVOXZNRGx0UzJaME4xWjNQVDBLTFMwdExTMUZUa1FnVUZWQ1RFbERJRXRGV1MwdExTMHRDZz09In19fX0=","integratedTime":1680651956,"logIndex":17126562,"logID":"c0d23d6ad406973f9559f3ba2d1ca01f84147d8ffc5b8445c224f98b9591801d"}}}}]







❯ sbsign --engine pkcs11 --cert sbtest.crt.pem --key 'pkcs11:object=sb-test;pin-value=0001password' test.efi 
warning: data remaining[110592 vs 121739]: gaps between PE/COFF sections?
warning: data remaining[110592 vs 121744]: gaps between PE/COFF sections?
Signing Unsigned original image


❯ cat openssl-pkcs11-engine.conf 
openssl_conf = openssl_init

[openssl_init]
engines = engine_section

[engine_section]
pkcs11 = pkcs11_section

[pkcs11_section]
engine_id = pkcs11
# dynamic_path is not required if you have installed
# the appropriate pkcs11 engines to your openssl directory
#dynamic_path = /path/to/engine_pkcs11.{so|dylib}
MODULE_PATH = /home/mheese/lib/yubihsm_pkcs11.so
# it is not recommended to use "debug" for production use
# also, you could just pick up the module settings through
# the PKCS11 configuration file with the environment variable
# YUBIHSM_PKCS11_CONF which will be used by the yubihsm pkcs11 module
# so no need for any init args
#INIT_ARGS = connector=http://yggdrasil:12345
#init = 0



❯ cat yubihsm_pkcs11.conf 
# This is a sample configuration file for the YubiHSM PKCS#11 module
# Uncomment the various options as needed

# URL of the connector to use. This can be a comma-separated list
# yggdrasil
#connector = http://192.168.87.197:12345
# rincewind
connector = http://192.168.87.199:12345

# Enables general debug output in the module
#
# debug

# Enables function tracing (ingress/egress) debug output in the module
#
# dinout

# Enables libyubihsm debug output in the module
#
# libdebug

# Redirects the debug output to a specific file. The file is created
# if it does not exist. The content is appended
#
# debug-file = /tmp/yubihsm_pkcs11_debug

# CA certificate to use for HTTPS validation. Point this variable to
# a file containing one or more certificates to use when verifying
# a peer. Currently not supported on Windows
#
# cacert = /tmp/cacert.pem

# Proxy server to use for the connector
# Currently not supported on Windows
#
# proxy = http://proxyserver.local.com:8080

# Timeout in seconds to use for the initial connection to the connector
# timeout = 5





OPENSSL_CONF environment variable must be set
YUBIHSM_PKCS11_CONF environment variable must be set
