# Changes

## v0.2.7

- Allowed setting resource requirements for the smi-adaptor
- Added ability to set `runAsUser` entry for the smi-adaptor
- Fixed `clusterDomain` config (it was being ignored)

## v0.2.6

This release adds `imagePullSecrets` support, for pulling images from private
docker registries.

## v0.2.5

This release just bumps the Helm chart version, which we had missed in the
previous releases.

## v0.2.4

This release fixes an issue where CLI flags were not being parsed.

## v0.2.3

Replaced `curlimages/curl` docker image in the `namespace-metadata` Job with
linkerd's `extension-init` image, to avoid all the OS luggage included in the
former, which generates CVE alerts.

## v0.2.2

This is a maintenance release which upgrades a number of dependencies.

## v0.2.1

This is a maintenance release which upgrades a number of dependencies and moves
the project onto Go 1.19.

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
