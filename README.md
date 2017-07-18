# socks52local

将socks5代理变成TCP代理。用来支持那些不支持socks5代理的程序。


# Build & Install

编译和安装就像普通的go程序那样，在命令行执行

`go get github.com/shellus/socks52local`

如无意外，`socks52local.exe`会出现在你的`GOPATH/bin`目录中


# Usage

使用起来就像这样

`socks52local.exe -x 127.0.0.1:1251 -l :22 -a 127.0.0.1:22`

