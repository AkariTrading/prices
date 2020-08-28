package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/akaritrading/prices/binance"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {

	err := binance.Init(time.Minute, "USDT", "BTC")
	if err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.DefaultLogger)
	r.Get("/{exchange}/priceStream/{symbol}", pricesWebsocket)
	r.Get("/{exchange}/orderbookPrice/{symbol}", orderbookPrice)

	http.ListenAndServe(":8080", r)
}

func orderbookPrice(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToLower(chi.URLParam(r, "symbol"))
	exchange := strings.ToLower(chi.URLParam(r, "exchange"))

	if exchange == "binance" {
		if !binance.CheckSymbol(symbol) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		price, _ := binance.OrderBookPrice(symbol)
		js, _ := json.Marshal(price)
		w.Write(js)
	}

	w.WriteHeader(http.StatusNotFound)
	return
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
		sub := stream.Subscribe(1)

		conn.SetCloseHandler(func(code int, text string) error {
			sub.Unsubscribe()
			return nil
		})

		for {
			val, _ := sub.Read()
			err = conn.WriteJSON(val)
			if err != nil {
				sub.Unsubscribe()
				return
			}
		}
	}

	w.WriteHeader(http.StatusNotFound)
	return
}
