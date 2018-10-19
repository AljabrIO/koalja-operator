# Copy the binaries into the image
FROM scratch
ARG GOARCH=amd64

WORKDIR /apps/
COPY bin/linux/${GOARCH}/manager .
COPY bin/linux/${GOARCH}/local_fs_service .
