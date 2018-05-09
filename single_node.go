package main

import (
    "crypto/tls"
    "crypto/x509"
    "io/ioutil"
    "log"
    "net/http"
    "net/url"
    "sync"
    "time"
)

func SingleNode(toCall string) []byte {
    responseChannel := make(chan *Response, *reqCntPerConnect*(*numConnections))

    benchTime := NewTimer()
    benchTime.Reset()
    //TODO check ulimit
    wg := &sync.WaitGroup{}

    tr := &http.Transport{
        DisableKeepAlives:     *disableKeepAlives,
        IdleConnTimeout:       time.Millisecond * time.Duration(*idleConnTimeout),
        ResponseHeaderTimeout: time.Millisecond * time.Duration(*respHeaderTimeout),
        MaxIdleConnsPerHost:   *maxIdleConnPerHost,
    }

    u, err := url.Parse(toCall)

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

    for i := 0; i < *numConnections; i++ {
        go StartClient(
            tr,
            toCall,
            *headers,
            *requestBody,
            *method,
            responseChannel,
            wg,
            *reqCntPerConnect,
        )
        wg.Add(1)
    }

    wg.Wait()

    result := CalcStats(
        responseChannel,
        benchTime.Duration(),
    )
    return result
}
