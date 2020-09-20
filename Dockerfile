FROM golang:1.14

WORKDIR /app
COPY . .
ARG GH_TOKEN
RUN go env -w GOPRIVATE=github.com/akaritrading/*
RUN git config --global url."https://${GH_TOKEN}:x-oauth-basic@github.com/".insteadOf "https://github.com/"
RUN go build -o main .

CMD ["/app/main"]