# Package `github.com/AljabrIO/koalja-operator/cmd`

This package and its sub-folders contains the program entry points
for all Koalja binaries.

Note that binaries are organized to perform multiple, but related roles.

For example there is one binary that contains all agents.
To run a specific agent, use a commandline argument.

```bash
# Example of running the link agent
agents link ...
# Example of running the pipeline agent
agents pipeline ...
```
