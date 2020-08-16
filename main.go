package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/websocket"
	"github.com/scarface382/prices/binance"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {

	err := binance.Init(time.Minute, "USDT")
	if err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.DefaultLogger)
	r.Get("/prices/{exchange}/{symbol}", pricesWebsocket)

	http.ListenAndServe(":8080", r)
}

func pricesWebsocket(w http.ResponseWriter, r *http.Request) {

	symbol := strings.ToLower(chi.URLParam(r, "symbol"))
	exchange := strings.ToLower(chi.URLParam(r, "exchange"))

	if exchange == "binance" {
		if !binance.CheckSymbol(symbol) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			return
		}

		defer conn.Close()

		stream := binance.PriceStream(symbol)
		sub := stream.Subscribe()

		conn.SetCloseHandler(func(code int, text string) error {
			stream.Unsubscribe(sub)
			return nil
		})

		for {
			err = conn.WriteJSON(<-sub)
			if err != nil {
				stream.Unsubscribe(sub)
				return
			}
		}
	}

	w.WriteHeader(http.StatusNotFound)
	return
}
