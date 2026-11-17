// based upon https://golang.cafe/blog/how-to-reuse-http-connections-in-go
package main

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptrace"
)

func get(traceCtx context.Context, url string) {
	req, err := http.NewRequestWithContext(traceCtx, http.MethodGet, "https://mastodon.zunda.ninja/health", nil)
	if err != nil {
		log.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		log.Fatal(err)
	}
	res.Body.Close()
}

func main() {
	clientTrace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) { log.Printf("conn was reused: %t", info.Reused) },
	}
	traceCtx := httptrace.WithClientTrace(context.Background(), clientTrace)

	url := "https://mastodon.zunda.ninja/health"
	get(traceCtx, url)
	get(traceCtx, url)
}
