FROM golang:1.14

WORKDIR /go/src/app
COPY . .

RUN git config --global url."https://${GH_TOKEN}:x-oauth-basic@github.com/".insteadOf "https://github.com/"
RUN go env -w GOPRIVATE=github.com/akaritrading/*
RUN go build

CMD ["./app"]