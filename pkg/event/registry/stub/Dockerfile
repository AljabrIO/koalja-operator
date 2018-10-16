# Copy the stub/registry into a thin image
FROM scratch
ARG GOARCH=amd64

WORKDIR /root/
COPY bin/linux/${GOARCH}/stub/registry .
ENTRYPOINT ["./registry"]
