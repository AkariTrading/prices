FROM golang:1.14

WORKDIR /go/src/app
COPY . .
ENV USER_EMAIL vaxkbihm@sharklasers.com
ENV USER_PASSWORD password

RUN go build

CMD ["./app"]