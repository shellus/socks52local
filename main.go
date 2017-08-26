package main

import (
	"net"
	"io"
	"github.com/shellus/pkg/logs"
	"context"
	"github.com/gamexg/proxyclient"
	"flag"
	"strings"
	"crypto/tls"
)

type Tunnel struct {
	conn net.Conn
	kcp  net.Conn
}

var (
	socksAddr string
	listenAddr string
	targetAddr string
)

func main() {
	flag.StringVar(&socksAddr, "x", "", `
 参数格式：允许使用 ?参数名1=参数值1&参数名2=参数值2指定参数
 例如：https://123.123.123.123:8088?insecureskipverify=true
     全体协议可选参数： upProxy=http://145.2.1.3:8080 用于指定代理的上层代理，即代理嵌套。默认值：direct://0.0.0.0:0000

 http 代理 http://123.123.123.123:8088
     可选功能： 用户认证功能。格式：http://user:password@123.123.123:8080
     可选参数：standardheader=false true表示 CONNNET 请求包含标准的 Accept、Accept-Encoding、Accept-Language、User-Agent等头。默认值：false

 https 代理 https://123.123.123.123:8088
     可选功能： 用户认证功能，同 http 代理。
     可选参数：standardheader=false 同上 http 代理
     可选参数：insecureskipverify=false true表示跳过 https 证书验证。默认false。
     可选参数：domain=域名 指定https验证证书时使用的域名，默认为 host:port
 socks4 代理 socks4://123.123.123.123:5050
     注意：socks4 协议不支持远端 dns 解析

 socks4a 代理 socks4a://123.123.123.123:5050

 socks5 代理 socks5://123.123.123.123:5050
     可选功能：用户认证功能。支持无认证、用户名密码认证，格式同 http 代理。

 ss 代理 ss://method:passowd@123.123.123:5050

 直连 direct://0.0.0.0:0000
     可选参数： LocalAddr=0.0.0.0:0 表示tcp连接绑定的本地ip及端口，默认值 0.0.0.0:0。
     可选参数： SplitHttp=false true 表示拆分 http 请求(分多个tcp包发送)，可以解决简单的运营商 http 劫持。默认值：false 。
              原理是：当发现目标地址为 80 端口，发送的内容包含 GET、POST、HTTP、HOST 等关键字时，会将关键字拆分到两个包在发送出去。
              注意： Web 防火墙类软件、设备可能会重组 HTTP 包，造成拆分无效。目前已知 ESET Smart Security 会造成这个功能无效，即使暂停防火墙也一样无效。
              G|ET /pa|th H|TTTP/1.0
              HO|ST:www.aa|dd.com
     可选参数： sleep=0  建立连接后延迟多少毫秒发送数据，配合 ttl 反劫持系统时建议设置为10置50。默认值 0 .
	`)

	flag.StringVar(&listenAddr, "l", "", "listen address e.g: :80")

	flag.StringVar(&targetAddr, "a", "", "target address e.g: 192.168.1.1:1080")

	flag.Parse()

	if listenAddr == "" {
		logs.Fatal("listenAddr(l) can't empty")
	}
	logs.Info("listen in: %s", listenAddr)

	if socksAddr == "" {
		socksAddr = "direct://0.0.0.0:0000"
	}
	if strings.Index(socksAddr, "://") == -1 {
		socksAddr = "socks5://" + socksAddr
	}

	logs.Info("use proxy: %s", socksAddr)


	if targetAddr == "" {
		logs.Fatal("targetAddr(a) %s invalid", targetAddr)
	}

	logs.Info("use target: %s", targetAddr)

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
		logs.Warning(err)
		return
	}
	defer tunn.Close()
	tunn.Work()
}

func NewTunnel(conn net.Conn) (*Tunnel, error) {

	proxyNet, err := proxyclient.NewProxyClient(socksAddr)
	if err != nil {
		logs.Fatal(err)
	}

	kcpConn, err := proxyNet.Dial("tcp", targetAddr)
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