FROM debian:latest
RUN apt update

RUN apt install apt-transport-https ca-certificates -y

COPY source.txt /etc/apt/sources.list

RUN apt update

RUN apt install wget -y

#  安装Go语言环境

RUN wget https://studygolang.com/dl/golang/go1.18.4.linux-amd64.tar.gz
RUN tar -xf go1.18.4.linux-amd64.tar.gz
RUN rm go1.18.4.linux-amd64.tar.gz

#  设置Go语言代理

RUN /go/bin/go env -w GOPROXY=https://goproxy.cn,direct

#  安装php环境

RUN apt install php -y

#  安装php插件

RUN apt install php-curl -y

#  安装g++环境

RUN apt install g++ -y

#  复制后端文件

RUN mkdir /base
COPY base /base
WORKDIR /base

#  编译后端文件
RUN /go/bin/go build