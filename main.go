package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
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

	r.Route("/{exchange}/history/{symbol}", func(newRoute chi.Router) {
		// newRoute.Use(middleware.Compress(5))
		newRoute.Get("/", priceHistory)

	})

	http.ListenAndServe(":8080", r)
}

func priceHistory(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	exchange := strings.ToLower(chi.URLParam(r, "exchange"))
	symbol := strings.ToLower(chi.URLParam(r, "symbol"))

	fmt.Println(symbol)

	start, err := strconv.ParseInt(r.URL.Query().Get("start"), 10, 64)
	if err != nil {
		start = 0
	}

	if exchange == "binance" {
		if !binance.CheckSymbol(symbol) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		hist, err := binance.GetSymbolHistory(symbol, start)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		bdy, err := json.Marshal(hist)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(bdy)
	}
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
