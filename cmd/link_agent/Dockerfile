# Copy the link_agent into a thin image
FROM scratch
ARG GOARCH=amd64

WORKDIR /root/
COPY bin/linux/${GOARCH}/link_agent .
ENTRYPOINT ["./link_agent"]
