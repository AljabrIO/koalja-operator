# Copy the controller-manager into a thin image
FROM scratch
ARG GOARCH=amd64

WORKDIR /root/
COPY bin/linux/${GOARCH}/manager .
ENTRYPOINT ["./manager"]
