# Package `github.com/AljabrIO/koalja-operator/pkg`

This package and its sub-folders contains main functions of the various Koalja
libraries & binaries.

## Directories

- [`agent`](./agent/) Package containing the implementation of link, task & pipeline agents.
- [`annotatedvalue`](./annotatedvalue/) Package containing the type definition for `AnnotatedValue` and
  related types & services.
- [`apis`](./apis/) Package containing (in various sub-packages) the Kubernetes APIs
  for Koalja custom resources.
- [`constants`](./constants/) Well know constants used in the entire product.
- [`controller`](./controller/) Package containing the pipeline operator.
- [`fs`](./fs/) Package containing filesystem service definition. Implementation and client
  are found in sub-packages.
- [`task`](./task/) Package containing custom task executors.
- [`tracking`](./tracking) Package containing tracking & statistics type & service definitions.
- [`util`](./util/) Package containing utility functions.
