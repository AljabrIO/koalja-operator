# Copy the local_fs_service into a thin image
FROM scratch
ARG GOARCH=amd64

WORKDIR /root/
COPY bin/linux/${GOARCH}/local_fs_service .
ENTRYPOINT ["./local_fs_service"]
