# The binary is cross-compiled on the build platform, so building for another
# architecture (--platform) needs no emulation.
FROM --platform=$BUILDPLATFORM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -o /komodo-exporter ./cmd/komodo-exporter

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /komodo-exporter /komodo-exporter
EXPOSE 9821
ENTRYPOINT ["/komodo-exporter"]
