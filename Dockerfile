FROM golang:1.17-alpine as BUILD
WORKDIR /app
RUN go get google.golang.org/protobuf/cmd/protoc-gen-go@v1.26 && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26 && \
    go get google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1
COPY . .
RUN go mod download
RUN mkdir out && go build -o out ./cmd/server

FROM alpine
RUN addgroup -S go && adduser -S go -G go
USER go:go
COPY --from=build out/* .
ENTRYPOINT ["server"]
