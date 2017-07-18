package main

import (
	"net"
	"io"
	"github.com/shellus/pkg/logs"
	"context"
	"github.com/gamexg/proxyclient"
	"flag"
	"fmt"
)

type Tunnel struct {
	conn net.Conn
	kcp  net.Conn
}

var (
	socksAddr string
	listenAddr string
	linkAddr string
)

func main() {
	flag.StringVar(&listenAddr, "l", "", "listen address e.g: :80")
	flag.StringVar(&socksAddr, "x", "", "socks address e.g: 127.0.0.1:1080")
	flag.StringVar(&linkAddr, "a", "", "target address e.g: 192.168.1.1:1080")
	flag.Parse()

	if listenAddr == "" {
		logs.Fatal("listen(l) %s invalid", listenAddr)
	}
	if socksAddr == "" {
		logs.Fatal("socksAddr(x) %s invalid", socksAddr)
	}
	if linkAddr == "" {
		logs.Fatal("linkAddr(a) %s invalid", linkAddr)
	}

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		panic(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {

	logs.Info("handleConn %s", conn.RemoteAddr().String())
	defer conn.Close()
	defer logs.Info("Conn close %s", conn.RemoteAddr().String())

	tunn, err := NewTunnel(conn)
	if err != nil {
		panic(err)
	}
	defer tunn.Close()
	tunn.Work()
}

func NewTunnel(conn net.Conn) (*Tunnel, error) {

	proxyNet, err := proxyclient.NewProxyClient(fmt.Sprintf("socks5://%s", socksAddr))
	if err != nil {
		logs.Fatal(err)
	}

	kcpConn, err := proxyNet.Dial("tcp", linkAddr)
	if err != nil {
		return nil, err
	}
	tun := &Tunnel{
		conn:conn,
		kcp:kcpConn,
	}
	return tun, nil
}
func (tun *Tunnel) Work() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tun.cp(ctx)
}

func (tun *Tunnel) Close() {
	tun.kcp.Close()
}

func (tun *Tunnel) cp(parentCtx context.Context) {
	defer tun.kcp.Close()
	defer tun.conn.Close()

	ctx, cancel := context.WithCancel(parentCtx)

	go func() {
		_, err := io.Copy(tun.kcp, tun.conn);
		if err != nil {
			logs.Debug(err)
		}
		cancel()
	}()
	go func() {
		_, err := io.Copy(tun.conn, tun.kcp);
		if err != nil {
			logs.Debug(err)
		}
		cancel()
	}()
	<-ctx.Done()
	if err := ctx.Err(); err != nil && context.Canceled != err {
		logs.Error(err)
	}
}