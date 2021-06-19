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
	localHost          = "localhost:8080"
	proxyHost          = ""
	proxyUser          = ""
	proxyAuthorization = ""
	remoteHost         = ""
)

func HandleRequest(clientConn net.Conn) {
	if proxyConn, err := net.Dial("tcp", proxyHost); err != nil {
		log.Fatal(err)
	} else {
		proxyauth := ""
		if proxyAuthorization != "" {
			proxyauth = fmt.Sprintf("Proxy-Authorization: %s", proxyAuthorization) + "\r\n"
		}
		request := fmt.Sprintf("CONNECT %s HTTP/1.0\r\n%s\r\n", remoteHost, proxyauth)
		proxyConn.Write([]byte(request))

		scanner := bufio.NewScanner(proxyConn)
		scanner.Scan()
		response := scanner.Text()
		if !strings.Contains(response, "200") {
			fmt.Println("err: " + response)
			proxyConn.Close()
			clientConn.Close()
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
	flag.StringVar(&proxyUser, "u", "", "username:password")
	flag.StringVar(&localHost, "p", "localhost:8080", "Proxy:port")
	flag.StringVar(&remoteHost, "r", "", "remote host:port")
	flag.StringVar(&proxyHost, "x", "10.1.16.8:8080", "Proxy:port")
	flag.Parse()

	proxyUrlString := ""
	if proxyUser != "" {
		proxyAuthorization = "Basic " + base64.StdEncoding.EncodeToString([]byte(proxyUser))
		proxyUrlString = fmt.Sprintf("http://%s@%s", strings.Replace(url.QueryEscape(proxyUser), "%3A", ":", 1), proxyHost)
	} else {
		proxyUrlString = fmt.Sprintf("http://%s", proxyHost)
	}
	proxyUrl, err := url.Parse(proxyUrlString)
	if err != nil {
		log.Fatal(err)
	}
	http.DefaultTransport = &http.Transport{Proxy: http.ProxyURL(proxyUrl)}
	listener, err := net.Listen("tcp", localHost)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer listener.Close()
	fmt.Println("Listening on", localHost)
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		go HandleRequest(conn)
	}
}
