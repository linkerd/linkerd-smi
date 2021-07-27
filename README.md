# linkerd-smi

[![Actions](https://github.com/linkerd/linkerd-smi/actions/workflows/integration_tests.yml/badge.svg)](https://github.com/linkerd/linkerd-smi/actions/workflows/integration_tests.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/linkerd/linkerd-smi)](https://goreportcard.com/report/github.com/linkerd/linkerd-smi)
[![GitHub license](https://img.shields.io/github/license/linkerd/linkerd-smi.svg)](LICENSE)

The Linkerd SMI extension helps users to have [SMI](https://smi-spec.io/) functionality
in [Linkerd](https://linkerd.io)-enabled Kubernetes clusters.

This repo consists of two components:

- `smi-adaptor`: Runs on your Kubernetes cluster, and transforms SMI
  resources into Linkerd native resources.
- `cli`: Runs locally or wherever you install the Linkerd CLI.

## Installation

### CLI

To install the CLI, run:

```bash
curl -sL https://linkerd.github.io/linkerd-smi/install | sh   
```

Alternatively, you can download the CLI directly via the
[releases page](https://github.com/linkerd/linkerd-smi/releases).

### Helm

To install the linkerd-smi Helm chart, run:

```bash
# add `linkerd-smi` helm repo
helm repo add l5d-smi https://linkerd.github.io/linkerd-smi

# install the chart
helm install l5d-smi/linkerd-smi --generate-name
```

## License

Copyright 2021 the Linkerd Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use
these files except in compliance with the License. You may obtain a copy of the
License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed
under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied. See the License for the
specific language governing permissions and limitations under the License.
