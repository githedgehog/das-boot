FROM gcr.io/distroless/static:nonroot
WORKDIR /

# upload the seeder binary to this docker image
COPY --chown=0:0 seeder /bin/

# the entrypoint is the seeder which should be a no-brainer
ENTRYPOINT [ "/bin/seeder" ]