FROM golang:1.7.3-alpine
MAINTAINER Ivan Gaas <ivan.gaas@gmail.com>

RUN apk -q --no-progress add --update --no-cache  git 

COPY apiserver.go /go/src/apiserver/

RUN cd /go/src/apiserver/ && go get ./ && go build ./

EXPOSE 80

ENTRYPOINT ["/go/bin/apiserver"]
