# Changes

## v0.1.0

linkerd-smi 0.1.0 is the first public release of the SMI extension
for Linkerd. This extension follows the [Linkerd's extension model](https://github.com/linkerd/linkerd2/blob/main/EXTENSIONS.md),
and ships with both a CLI and a Helm Chart.

The `smi-adaptor` is the main component of this extension. It is a Kubernetes
controller that listens for `TrafficSplit` objects and converts them into
a new corresponding `ServiceProfile` object, or updates the existing one
if it already exists.
