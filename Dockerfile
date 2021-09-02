FROM golang:1.17 as BUILD
RUN apt update && apt install unzip
WORKDIR /app
ENV PROTOC_ZIP=protoc-3.17.3-linux-x86_64.zip
RUN wget https://github.com/protocolbuffers/protobuf/releases/download/v3.17.3/protoc-3.17.3-linux-x86_64.zip && \
    unzip *.zip && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1
COPY . .
RUN go mod download
RUN ./bin/protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative transport/http.proto
RUN mkdir out && go build -o out ./cmd/server
RUN openssl req -x509 -newkey rsa:4096 -days 365 -nodes -keyout ca-key.pem -out ca-cert.pem -subj "/C=US/" && \
    openssl req -newkey rsa:4096 -nodes -keyout server-key.pem -out server-req.pem -subj "/C=FR/" && \
    openssl x509 -req -in server-req.pem -days 60 -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial -out server-cert.pem -extfile server-ext.cnf

ENTRYPOINT ["/app/out/server"]
