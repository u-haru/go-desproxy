package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var (
	localport          = 8080
	proxyHost          = ""
	remoteHost         = ""
	proxyAuthorization = ""
)

func HandleRequest(clientConn net.Conn) {
	if proxyConn, err := net.Dial("tcp", proxyHost); err != nil {
		log.Fatal(err)
	} else {
		url, err := url.Parse("http://" + remoteHost)
		if err != nil {
			log.Fatal(err)
		}
		addr, err := net.ResolveIPAddr("ip4", url.Hostname())
		if err != nil {
			log.Fatal(err)
		}
		req, err := http.NewRequest("CONNECT", "https://"+remoteHost, nil)
		if err != nil {
			log.Fatal(err)
		}
		req.Host = fmt.Sprintf("%s", addr.String()+":"+url.Port())
		req.Header.Set("Proxy-Authorization", proxyAuthorization)
		req.Write(proxyConn)
		loop := true
		br := bufio.NewReader(proxyConn)
		for loop {
			b, err := br.ReadByte()
			if b == '\n' {
				loop = false
			}
			if err != nil {
				log.Fatal(err)
			}
		}
		go func() {
			io.Copy(clientConn, proxyConn)
			proxyConn.Close()
		}()
		go func() {
			io.Copy(proxyConn, clientConn)
			clientConn.Close()
		}()
	}
}

func main() {
	_proxyUser := flag.String("u", "", "username:password")
	_localport := flag.Int("p", 8080, "local port")
	_remoteHost := flag.String("r", "", "remote host:port")
	_proxyHost := flag.String("x", "10.1.16.8:8080", "Proxy:port")
	flag.Parse()
	localport = *_localport
	remoteHost = *_remoteHost
	proxyHost = *_proxyHost

	proxyAuthorization = "Basic " + base64.StdEncoding.EncodeToString([]byte(*_proxyUser))
	proxyUrlString := fmt.Sprintf("http://%s@%s", strings.Replace(url.QueryEscape(*_proxyUser), "%3A", ":", 1), proxyHost)
	proxyUrl, err := url.Parse(proxyUrlString)
	if err != nil {
		log.Fatal(err)
	}
	http.DefaultTransport = &http.Transport{Proxy: http.ProxyURL(proxyUrl)}
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", localport))
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer listener.Close()
	fmt.Println("Listening on localhost:")
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		go HandleRequest(conn)
	}
}
