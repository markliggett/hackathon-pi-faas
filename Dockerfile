FROM golang:alpine as builder

RUN apk update && apk add --no-cache git
RUN adduser -D -g '' appuser

ADD go /go
WORKDIR /go/src
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM scratch
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /go/src/main /main
USER appuser
CMD ["/main"]