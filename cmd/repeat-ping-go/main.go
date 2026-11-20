/*
Usage:

  INTERVAL=60 TARGET=https://zunda.ninja repeat-ping-go

repeatedly sends requests to the TARGET with the INTERVAL
and prints metrics
*/

package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultInterval = 600
	defaultUrl      = "https://zunda.ninja"
	defaultSamples  = 0
	defaultTimeout  = 60
)

type transport struct {
	current *http.Request

	StartedAt   time.Time
	CompletedAt time.Time
	TimeTotal   float64

	Url            string
	Errmsg         string
	LocalIp        string
	LocalPort      string
	RemoteIp       string
	RemotePort     string
	ResponseCode   int
	SrcIp          string
	ReqOnConnCount int

	afterPrevReq []string
	AfterPrevReq string
}

func (t *transport) GetConn(_ string) {
	t.afterPrevReq = append(t.afterPrevReq, "GetConn")
}

func (t *transport) GotConn(info httptrace.GotConnInfo) {
	t.LocalIp, t.LocalPort, _ = net.SplitHostPort(info.Conn.LocalAddr().String())
	t.RemoteIp, t.RemotePort, _ = net.SplitHostPort(info.Conn.RemoteAddr().String())
	t.afterPrevReq = append(t.afterPrevReq, "GotConn")
	if info.Reused {
		t.ReqOnConnCount += 1
	} else {
		t.ReqOnConnCount = 1
	}
}

func (t *transport) PutIdleConn(_ error) {
	t.afterPrevReq = append(t.afterPrevReq, "PutIdleConn")
}

func (t *transport) GotFirstResponseByte() {
	t.afterPrevReq = append(t.afterPrevReq, "GotFirstResponseByte")
}

func (t *transport) DNSStart(_ httptrace.DNSStartInfo) {
	t.afterPrevReq = append(t.afterPrevReq, "DNSStart")
}

func (t *transport) DNSDone(_ httptrace.DNSDoneInfo) {
	t.afterPrevReq = append(t.afterPrevReq, "DNSDone")
}

func (t *transport) ConnectStart(_, _ string) {
	t.afterPrevReq = append(t.afterPrevReq, "ConnectStart")
}

func (t *transport) ConnectDone(_, _ string, _ error) {
	t.afterPrevReq = append(t.afterPrevReq, "ConnectDone")
}

func (t *transport) TLSHandshakeStart() {
	t.afterPrevReq = append(t.afterPrevReq, "TLSHandshakeStart")
}

func (t *transport) TLSHandshakeDone(_ tls.ConnectionState, _ error) {
	t.afterPrevReq = append(t.afterPrevReq, "TLSHandshakeDone")
}

func (t *transport) WroteHeaders() {
	t.afterPrevReq = append(t.afterPrevReq, "WroteHeaders")
}

func (t *transport) WroteRequest(_ httptrace.WroteRequestInfo) {
	t.afterPrevReq = append(t.afterPrevReq, "WroteRequest")
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.current = req
	return http.DefaultTransport.RoundTrip(req)
}

func (t *transport) log() {
	e := json.NewEncoder(os.Stdout)
	t.TimeTotal = t.CompletedAt.Sub(t.StartedAt).Seconds()
	t.AfterPrevReq = strings.Join(t.afterPrevReq, ",")
	err := e.Encode(t)
	if err != nil {
		fmt.Println(os.Stderr, err.Error())
	}
	t.afterPrevReq = nil
	os.Stdout.Sync()
}

func get(req *http.Request, transport *transport, timeout time.Duration) {
	transport.Url = req.URL.String()
	transport.StartedAt = time.Now()
	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
	res, err := client.Do(req)
	transport.CompletedAt = time.Now()
	if err != nil {
		transport.Errmsg = err.Error()
		goto Log
	}
	defer res.Body.Close()
	transport.ResponseCode = res.StatusCode
	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		transport.Errmsg = err.Error()
		goto Log
	}
Log:
	transport.log()
}

func repeatGet(
	interval time.Duration,
	req *http.Request,
	transport *transport,
	samples int,
	timeout time.Duration,
) {
	for {
		get(req, transport, timeout)
		if samples > 0 {
			samples--
			if samples == 0 {
				break
			}
		}
		time.Sleep(interval)
	}
}

func srcIp() string {
	res, err := http.Get("https://4.ipaddr.io/ip")
	if err != nil {
		return err.Error()
	}
	defer res.Body.Close()
	srcIp, err := io.ReadAll(res.Body)
	if err != nil {
		return err.Error()
	}
	return string(srcIp)
}

var optInterval float64
var optUrl string
var optTimeout float64
var samples int

func init() {
	flag.Float64Var(
		&optInterval,
		"i",
		defaultInterval,
		"interval between requests (sec)",
	)
	flag.StringVar(&optUrl, "t", defaultUrl, "target URL")
	flag.Float64Var(
		&optTimeout,
		"l",
		defaultTimeout,
		"time limit to receive HTTP response (sec)",
	)
	flag.IntVar(
		&samples,
		"s",
		defaultSamples,
		"number of samples (0 for inifinte)",
	)
}

func main() {
	flag.Parse()

	var interval float64
	var url string
	var timeout float64

	i := os.Getenv("INTERVAL")
	if i != "" {
		interval, _ = strconv.ParseFloat(i, 64)
	}
	if interval == 0 {
		interval = optInterval
	}

	url = os.Getenv("TARGET")
	if url == "" {
		url = optUrl
	}

	l := os.Getenv("TIMEOUT")
	if l != "" {
		timeout, _ = strconv.ParseFloat(l, 64)
	}
	if timeout == 0 {
		timeout = optTimeout
	}

	t := &transport{}
	t.SrcIp = srcIp()
	req, _ := http.NewRequest("GET", url, nil)
	trace := &httptrace.ClientTrace{
		GetConn:              t.GetConn,
		GotConn:              t.GotConn,
		PutIdleConn:          t.PutIdleConn,
		GotFirstResponseByte: t.GotFirstResponseByte,
		DNSStart:             t.DNSStart,
		DNSDone:              t.DNSDone,
		ConnectStart:         t.ConnectStart,
		ConnectDone:          t.ConnectDone,
		TLSHandshakeStart:    t.TLSHandshakeStart,
		TLSHandshakeDone:     t.TLSHandshakeDone,
		WroteHeaders:         t.WroteHeaders,
		WroteRequest:         t.WroteRequest,
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	repeatGet(
		time.Duration(interval)*time.Second,
		req,
		t, samples,
		time.Duration(timeout)*time.Second,
	)
}
