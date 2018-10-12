# Copy the task_agent into a thin image
FROM scratch
ARG GOARCH=amd64

WORKDIR /root/
COPY bin/linux/${GOARCH}/task_agent .
ENTRYPOINT ["./task_agent"]
