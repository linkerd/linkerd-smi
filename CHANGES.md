# Changes

## v0.2.0

This release adds the `TrafficSplit` (`v1alpha1` and `v1alpha2`) CRD into the
extension. This also includes improvements around the non-default namespace
creation in helm, along with the controller to emit events while processing SMI
resources.

This version has compatibility with Linkerd starting from `edge-21.12.2` versions,
to prevent race conditions during the CRD install.

## v0.1.0

linkerd-smi 0.1.0 is the first public release of the SMI extension
for Linkerd. This extension follows the [Linkerd's extension model](https://github.com/linkerd/linkerd2/blob/main/EXTENSIONS.md),
and ships with both a CLI and a Helm Chart.

The `smi-adaptor` is the main component of this extension. It is a Kubernetes
controller that listens for `TrafficSplit` objects and converts them into
a new corresponding `ServiceProfile` object, or updates the existing one
if it already exists.
