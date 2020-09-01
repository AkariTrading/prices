package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/akaritrading/prices/exchanges/binance"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {

	err := binance.Init("TRY", "EUR")
	if err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.DefaultLogger)
	r.Get("/{exchange}/priceStream/{symbol}", pricesWebsocket)
	r.Get("/{exchange}/orderbookPrice/{symbol}", orderbookPrice)

	r.Route("/{exchange}/priceData", func(newRoute chi.Router) {
		newRoute.Use(middleware.Compress(5))
		newRoute.Get("/", priceData)

	})

	http.ListenAndServe(":8080", r)
}

func priceData(w http.ResponseWriter, r *http.Request) {
	fmt.Println("price data")
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("yo what up"))
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
