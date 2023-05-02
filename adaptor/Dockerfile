ARG BUILDPLATFORM=linux/amd64

FROM --platform=$BUILDPLATFORM golang:1.19-alpine as golang
WORKDIR /linkerd-smi-build
COPY go.mod go.sum ./
RUN go mod download
ARG TARGETARCH
COPY adaptor adaptor
COPY pkg pkg
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH  go build -o /out/smi-adaptor -tags production -mod=readonly -ldflags "-s -w" ./adaptor

FROM scratch
COPY --from=golang /out/smi-adaptor /smi-adaptor
ENTRYPOINT ["/smi-adaptor"]
