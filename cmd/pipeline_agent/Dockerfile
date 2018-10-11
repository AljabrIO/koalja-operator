# Copy the pipeline_agent into a thin image
FROM scratch
ARG GOARCH=amd64

WORKDIR /root/
COPY bin/linux/${GOARCH}/pipeline_agent .
ENTRYPOINT ["./pipeline_agent"]
