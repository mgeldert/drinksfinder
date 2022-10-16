FROM golang:1.16 as builder
WORKDIR /workspace
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -o apiserver *.go

FROM gcr.io/distroless/static-debian10:nonroot
WORKDIR /
COPY --from=builder /workspace/apiserver .
USER 65532:65532

ENTRYPOINT ["/apiserver"]
