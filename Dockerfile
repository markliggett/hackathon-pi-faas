FROM golang as builder

ADD go/src /go/src
RUN cd /go/src && go get github.com/opentracing-contrib/examples/go
RUN go get github.com/prometheus/client_golang/prometheus
RUN go get github.com/prometheus/client_golang/prometheus/promauto
RUN go get github.com/prometheus/client_golang/prometheus/promhttp
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM scratch
COPY --from=builder /go/src/main /main
CMD ["/main"]