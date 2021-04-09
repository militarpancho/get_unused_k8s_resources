FROM golang:1.16.3-alpine3.13
COPY ./ /go/src
WORKDIR /go/src
RUN go mod tidy && go build

FROM alpine:3.13

COPY --from=0 /go/src/get_unused_k8s_resources /go/bin/get_unused_k8s_resources

CMD ["/go/bin/get_unused_k8s_resources"]
