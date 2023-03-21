FROM ubuntu:22.04 AS builder
RUN apt-get update && apt-get install -y ca-certificates tzdata

FROM scratch

# This copies the trusted root CA certificates store from the builder image to
# our resulting image. This is necessary for TLS certificate verification to
# work with any public certificate. The underlying implementation for Linux
# simply relies on this file to be there. Technically, we do not really need
# this as we are not making use of any 3rd party resources which would require
# us to validate an external certificate. However, this is just good practice.
# See this source file for details:
# https://golang.org/src/crypto/x509/root_linux.go
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# There used to be some Golang versions which had issues with resolving time
# correctly if *both* time zoneinfo files *and* /etc/localtime were missing.
# This might not be the case anymore, but this hardens this for sure.
# See this source file for details (function `initLocal()`):
# https://golang.org/src/time/zoneinfo_unix.go
COPY --from=builder /etc/localtime /etc/localtime

# upload the seeder binary to this docker image
ADD seeder /bin/seeder

# ensure we run as a normal user in there
USER 1042:1042

# the entrypoint is the seeder which should be a no-brainer
ENTRYPOINT [ "/bin/seeder" ]