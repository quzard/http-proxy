package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"
)

func main() {
	l, err := net.Listen("tcp", ":1234")
	if err != nil {
		panic(err)
	}

	for {
		client, err := l.Accept()
		if err != nil {
			log.Panic(err)
		}
		go handleProxyRequest(client)
	}
}

func handleProxyRequest(client net.Conn) {
	buffer := make([]byte, 1024)

	n, err := client.Read(buffer)
	if err != nil {
		log.Panic(err)
		return
	}
	fmt.Println(string(buffer))
	s := strings.Split(string(buffer), " ")
	method, host := s[0], s[1]
	if method == "CONNECT" {
		go handleHttpsProxy(client, host)
	} else {
		go handleHttpProxy(client, host, buffer, n)
	}
}

func handleHttpProxy(client net.Conn, host string, buffer []byte, n int) {
	//GET http://www.google.com/ HTTP/1.1
	//Host: www.google.com
	//User-Agent: curl/7.78.0
	//Accept: */*
	//Proxy-Connection: Keep-Alive
	hostPortURL, err := url.Parse(host)
	if strings.Index(hostPortURL.Host, ":") == -1 { //host不带端口， 默认80
		host = hostPortURL.Host + ":80"
	} else {
		host = hostPortURL.Host
	}
	fmt.Println(host)
	server, err := net.Dial("tcp", host)
	if err != nil {
		log.Panic(err)
		return
	}
	_, err = server.Write(buffer[:n])
	if err != nil {
		log.Panic(err)
		return
	}
	proxy(client, server)
}
func handleHttpsProxy(client net.Conn, host string) {
	//CONNECT www.google.com:443 HTTP/1.1
	//Host: www.google.com:443
	//User-Agent: curl/7.78.0
	//Proxy-Connection: Keep-Alive

	fmt.Println("https:    ", host)
	_, err := fmt.Fprint(client, "HTTP/1.1 200 Connection established\r\n\r\n")
	if err != nil {
		log.Panic(err)
		return
	}

	server, err := net.Dial("tcp", host)
	if err != nil {
		log.Panic(err)
		return
	}
	proxy(client, server)
}

func proxy(client, server net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := io.Copy(server, client)
		if err != nil {
			log.Panic(err)
			return
		}
	}()
	go func() {
		defer wg.Done()
		_, err := io.Copy(client, server)
		if err != nil {
			log.Panic(err)
			return
		}
	}()
	wg.Wait()
}
