ARG BUILDPLATFORM=linux/amd64

# Precompile key slow-to-build dependencies
FROM --platform=$BUILDPLATFORM golang:1.16.4-alpine as go-deps
WORKDIR /linkerd-smi-build
COPY go.mod go.sum ./
RUN go mod download
ARG TARGETARCH

## compile controller service
FROM go-deps as golang
WORKDIR /linkerd-smi-build
COPY adaptor adaptor
COPY pkg pkg

ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -o /out/smi-adaptor -tags production -mod=readonly -ldflags "-s -w" ./adaptor

## package runtime
FROM scratch
ARG LINKERD_VERSION
COPY --from=golang /out/smi-adaptor /smi-adaptor

ENTRYPOINT ["/smi-adaptor"]