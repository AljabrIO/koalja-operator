# Copy the binaries into the image
FROM alpine
ARG GOARCH=amd64

RUN apk add ca-certificates

WORKDIR /apps/
COPY bin/linux/${GOARCH}/services .
