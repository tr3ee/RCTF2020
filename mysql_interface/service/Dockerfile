FROM golang:latest as builder
WORKDIR /app
COPY ./go.mod .
COPY ./go.sum .
COPY ./vendor ./vendor
COPY ./mysql_interface.go .
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go build -mod vendor -o mysql_interface .


FROM debian:buster-slim
MAINTAINER tr3e

RUN apt-get update
RUN apt-get upgrade -y

RUN useradd -m ctf

WORKDIR /app

# set up mysql_interface service
COPY ./assets ./assets
COPY ./templates ./templates
COPY --from=builder /app/mysql_interface .

RUN chown -R root:ctf /app && \
    chmod -R 750 /app

# start the game!

EXPOSE 8081

CMD ["/app/mysql_interface"]