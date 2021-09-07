package proxy

import (
	"io"
	"local-webhook-tester/transport"
	"local-webhook-tester/util"
	"net/http"
	"net/url"
	"strings"
)

func deserializeGrpcHttpResponse(request *http.Request, response *transport.HttpResponse) *http.Response {
	responseHeader := http.Header{}
	for _, header := range response.Headers {
		split := strings.Split(header, ":")
		key := split[0]
		val := split[1][1:]

		responseHeader.Add(key, val)
	}

	httpResponse := &http.Response{
		StatusCode:       int(response.ResponseCode),
		Proto:            request.Proto,
		ProtoMajor:       request.ProtoMajor,
		ProtoMinor:       request.ProtoMinor,
		Header:           responseHeader,
		Body:             io.NopCloser(strings.NewReader(response.Body)),
		ContentLength:    int64(len(response.Body)),
		TransferEncoding: request.TransferEncoding,
		TLS:              request.TLS,
	}
	return httpResponse
}

func serializeHttpRequest(request *http.Request) (*transport.HttpRequest, error) {
	path := request.URL.Path
	bodyString, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	headers := util.SerializeHeader(request.Header)

	relativeUrl, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	return &transport.HttpRequest{
		Method:   request.Method,
		Path:     relativeUrl.Path,
		Body:     string(bodyString),
		Headers:  headers,
		RawQuery: request.URL.RawQuery,
	}, nil
}

func deserializeHttpResponseToResponseWriter(writer http.ResponseWriter, response *http.Response) error {
	for key, values := range response.Header {
		for _, val := range values {
			writer.Header().Add(key, val)
		}
	}

	writer.WriteHeader(response.StatusCode)
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	_, err = writer.Write(responseBody)
	return err
}
