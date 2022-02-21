FROM golang:1.17.0-alpine3.14
ENV GO111MODULE=on
WORKDIR /usr/src/app
COPY . .
RUN go mod download
RUN go build -o /usr/local/bin/pod-controller cmd/pod-controller/main.go
ENTRYPOINT [ "/usr/local/bin/pod-controller" ]

