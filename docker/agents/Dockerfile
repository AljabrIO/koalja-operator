# Copy the binaries into the image
FROM scratch
ARG GOARCH=amd64

WORKDIR /apps/
COPY bin/linux/${GOARCH}/pipeline_agent .
COPY bin/linux/${GOARCH}/task_agent .
COPY bin/linux/${GOARCH}/link_agent .
