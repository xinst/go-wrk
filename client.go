package main

import (
    "crypto/tls"
    "crypto/x509"
    "io/ioutil"
    "log"
    "net/http"
    "net/url"
    "strings"
    "sync"
    "time"
)

func StartClient(url_, heads, requestBody string, meth string, dka bool, responseChan chan *Response, waitGroup *sync.WaitGroup, reqCnt int) {
    defer waitGroup.Done()

    tr := &http.Transport{
        DisableKeepAlives:     dka,
        IdleConnTimeout:       time.Millisecond * time.Duration(*idleConnTimeout),
        ResponseHeaderTimeout: time.Millisecond * time.Duration(*respHeaderTimeout),
    }

    u, err := url.Parse(url_)

    if err == nil && u.Scheme == "https" {
        var tlsConfig *tls.Config
        if *insecure {
            tlsConfig = &tls.Config{
                InsecureSkipVerify: true,
            }
        } else {
            // Load client cert
            cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
            if err != nil {
                log.Fatal(err)
            }

            // Load CA cert
            caCert, err := ioutil.ReadFile(*caFile)
            if err != nil {
                log.Fatal(err)
            }
            caCertPool := x509.NewCertPool()
            caCertPool.AppendCertsFromPEM(caCert)

            // Setup HTTPS client
            tlsConfig = &tls.Config{
                Certificates: []tls.Certificate{cert},
                RootCAs:      caCertPool,
            }
            tlsConfig.BuildNameToCertificate()
        }
        tr.TLSClientConfig = tlsConfig
        tr.TLSHandshakeTimeout = time.Millisecond * time.Duration(*tlsHandshakeTimeout)
    }

    reqTimes := 0
    timer := NewTimer()
    for {
        requestBodyReader := strings.NewReader(requestBody)
        req, _ := http.NewRequest(meth, url_, requestBodyReader)
        sets := strings.Split(heads, "\n")

        //Split incoming header string by \n and build header pairs
        for i := range sets {
            split := strings.SplitN(sets[i], ":", 2)
            if len(split) == 2 {
                req.Header.Set(split[0], split[1])
            }
        }

        timer.Reset()

        resp, err := tr.RoundTrip(req)

        respObj := &Response{}

        if err != nil {
            respObj.Error = true
            log.Println(err.Error())
            respObj.ErrMsg = err.Error()
        } else {
            if resp.ContentLength < 0 { // -1 if the length is unknown
                data, err := ioutil.ReadAll(resp.Body)
                if err == nil {
                    respObj.Size = int64(len(data))
                } else {
                    respObj.ErrMsg = err.Error()
                }
            } else {
                respObj.Size = resp.ContentLength
            }
            respObj.StatusCode = resp.StatusCode
            resp.Body.Close()
        }

        respObj.Duration = timer.Duration()
        reqTimes++

        responseChan <- respObj

        if reqTimes >= reqCnt {
            break
        }
    }
}
