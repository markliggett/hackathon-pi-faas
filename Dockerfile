FROM golang as builder

ADD go /go
RUN cd /go/src && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM scratch
COPY --from=builder /go/src/main /main
CMD ["/main"]