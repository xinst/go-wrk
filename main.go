package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "os"
    "runtime"
)

type Config struct {
    Port  string
    Nodes []string
}

var (
    numThreads          = flag.Int("t", 1, "the numbers of threads used")
    method              = flag.String("m", "GET", "the http request method")
    requestBody         = flag.String("b", "", "the http requst body")
    requestBodyFile     = flag.String("p", "", "the http requst body data file")
    numConnections      = flag.Int("c", 100, "the max numbers of connections used")
    reqCntPerConnect    = flag.Int("n", 1000, "request count per connection")
    disableKeepAlives   = flag.Bool("k", true, "if keep-alives are disabled")
    idleConnTimeout     = flag.Int("idleTimeout", 10000, "the idle connection time out, default is 60000ms(60s)")
    dist                = flag.String("d", "", "dist mode")
    configFile          = flag.String("f", "", "json config file")
    config              Config
    target              string
    headers             = flag.String("H", "User-Agent: go-wrk 0.1 bechmark\nContent-Type: application/json;charset=utf-8", "the http headers sent separated by '\\n'")
    certFile            = flag.String("cert", "someCertFile", "A PEM eoncoded certificate file.")
    keyFile             = flag.String("key", "someKeyFile", "A PEM encoded private key file.")
    caFile              = flag.String("CA", "someCertCAFile", "A PEM eoncoded CA's certificate file.")
    insecure            = flag.Bool("i", false, "TLS checks are disabled")
    tlsHandshakeTimeout = flag.Int("tlsTimeOut", 10000, "TLS handshake timeout, in ms, default is 10000ms(10s) ")
    respHeaderTimeout   = flag.Int("respTimeOut", 10000, "response header timeout, in ms, default is 10000ms(10s)")
)

func init() {
    flag.Parse()
    target = os.Args[len(os.Args)-1]
    if *configFile != "" {
        readConfig()
    }
    runtime.GOMAXPROCS(*numThreads)
}

func readConfig() {
    configData, err := ioutil.ReadFile(*configFile)
    if err != nil {
        fmt.Println(err)
        panic(err)
    }
    err = json.Unmarshal(configData, &config)
    if err != nil {
        fmt.Println(err)
        panic(err)
    }
}

func setRequestBody() {
    // requestBody has been setup and it has highest priority
    if *requestBody != "" {
        return
    }

    if *requestBodyFile == "" {
        return
    }

    // requestBodyFile has been setup
    data, err := ioutil.ReadFile(*requestBodyFile)
    if err != nil {
        fmt.Println(err)
        panic(err)
    }
    body := string(data)
    requestBody = &body
}

func main() {
    setRequestBody()
    switch *dist {
    case "m":
        MasterNode()
    case "s":
        SlaveNode()
    default:
        SingleNode(target)
    }
}
