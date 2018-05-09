package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "sort"
    "strings"
    "sync"
    "time"
)

// RateAttack ...
func RateAttack() {
    rate := *reqRate
    du := time.Second * time.Duration(*dur)
    var wg sync.WaitGroup
    ticks := make(chan time.Time)

    tr := mkTransport(target)

    responseChannel := make(chan *Response, 100)

    go func() {
        defer close(responseChannel)
        defer wg.Wait()
        defer close(ticks)
        interval := 1e9 / rate
        hits := rate * uint64(du.Seconds())
        began, done := time.Now(), uint64(0)
        for {
            now, next := time.Now(), began.Add(time.Duration(done*interval))
            time.Sleep(next.Sub(now))
            select {
            case ticks <- max(next, now):
                if done++; done == hits {
                    return
                }
            default: // all workers are blocked. start one more and try again
                wg.Add(1)
                go attack(tr, target, *headers, *requestBody, *method, ticks, responseChannel, &wg)
            }
        }
    }()

    calcReqStats(responseChannel, rate, du.Nanoseconds()/1000)
}

func max(a, b time.Time) time.Time {
    if a.After(b) {
        return a
    }
    return b
}

func attack(tr *http.Transport,
    reqURL, heads, requestBody, meth string,
    ticks <-chan time.Time,
    responseChan chan *Response,
    waitGroup *sync.WaitGroup) {
    defer waitGroup.Done()

    for tm := range ticks {

        requestBodyReader := strings.NewReader(requestBody)
        req, _ := http.NewRequest(meth, reqURL, requestBodyReader)
        sets := strings.Split(heads, "\n")

        //Split incoming header string by \n and build header pairs
        for i := range sets {
            split := strings.SplitN(sets[i], ":", 2)
            if len(split) == 2 {
                req.Header.Set(split[0], split[1])
            }
        }

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

        respObj.Duration = time.Since(tm).Nanoseconds() / 1000
        responseChan <- respObj
    }
}

func calcReqStats(responseChannel chan *Response, rate uint64, duration int64) []byte {

    stats := &Stats{
        Url:         target,
        Connections: *numConnections,
        Threads:     *numThreads,
        Duration:    float64(duration),
        AvgDuration: float64(duration),
        Rate:        rate,
    }

    i := 0

FORLOOP:
    for {
        select {
        case res, ok := <-responseChannel:
            if !ok {
                break FORLOOP
            }
            switch {
            case res.StatusCode < 200:
                // error
            case res.StatusCode < 300:
                stats.Resp200++
            case res.StatusCode < 400:
                stats.Resp300++
            case res.StatusCode < 500:
                stats.Resp400++
            case res.StatusCode < 600:
                stats.Resp500++
            }

            stats.Sum += float64(res.Duration)
            stats.Times = append(stats.Times, int(res.Duration))
            i++

            stats.Transfered += res.Size

            if res.Error {
                stats.Errors++
            }
        }
    }
    stats.TotalCall = uint64(i)
    sort.Ints(stats.Times)

    PrintStats(stats)
    b, err := json.Marshal(&stats)
    if err != nil {
        fmt.Println(err)
    }
    return b
}
