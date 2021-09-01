Local Webhook Tester
=====

This tool is meant for testing local application behavior against external HTTP
calls that cannot be generated locally.

It uses bidirectional GRPC messages to pass along HTTP requests to a local
server and respond to the original caller with the response from the local server.

[local-server] ←http→ [grpc client] ←grpc→ [grpc server] ←http→ [http client]

Limitations
- Only one request at a time can be processed behind the grpc server. The server
can queue HTTP requests normally
- There's a 5-minute timeout on http requests to the grpc server. This can be changed