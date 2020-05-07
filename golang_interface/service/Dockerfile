FROM golang:latest as builder
WORKDIR /app
COPY ./go.mod .
COPY ./go.sum .
COPY ./vendor ./vendor
COPY ./golang_interface.go .
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go build -mod vendor -o golang_interface .


FROM golang:latest
MAINTAINER tr3e

RUN apt-get update
RUN apt-get upgrade -y

# set up sandbox env
RUN useradd -m ctf

WORKDIR /home/ctf

RUN cp -R /lib* /home/ctf && \
    cp -R /usr/lib* /home/ctf && \
    mkdir /home/ctf/builds

RUN mkdir /home/ctf/dev && \
    mknod /home/ctf/dev/null c 1 3 && \
    mknod /home/ctf/dev/zero c 1 5 && \
    mknod /home/ctf/dev/random c 1 8 && \
    mknod /home/ctf/dev/urandom c 1 9 && \
    chmod 666 /home/ctf/dev/*

RUN mkdir /home/ctf/bin && \
    cp /bin/sh /home/ctf/bin && \
    cp /bin/ls /home/ctf/bin && \
    cp /bin/cat /home/ctf/bin

COPY ./flag /home/ctf/flag

RUN chown -R root:ctf /home/ctf && \
    chmod -R 750 /home/ctf && \
    chmod 740 /home/ctf/flag && \
    chmod 710 /home/ctf/builds

# set up golang_interface service
WORKDIR /app

COPY ./assets ./assets
COPY ./templates ./templates
COPY --from=builder /app/golang_interface .
RUN mkdir /uploads

# start the game!


EXPOSE 8080

CMD ["/app/golang_interface"]