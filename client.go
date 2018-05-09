package main

import (
    "io/ioutil"
    "log"
    "net/http"
    "strings"
    "sync"
)

func StartClient(tr *http.Transport, url_, heads, requestBody string, meth string, responseChan chan *Response, waitGroup *sync.WaitGroup, reqCnt int) {
    defer waitGroup.Done()

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
