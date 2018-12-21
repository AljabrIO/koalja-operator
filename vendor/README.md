# Vendor

This folder contains external go packages that are being used in Koalja.

It is important that this vendor directory is "flattened" (no recursive vendor directories).

Once the Kubernetes libraries & `kubebuilder` can cope with it, this directory
should be replaced with proper use of go modules.
