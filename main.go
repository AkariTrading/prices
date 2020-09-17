package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/akaritrading/libs/middleware"
	"github.com/akaritrading/libs/util"
	"github.com/akaritrading/prices/exchanges/binance"
	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {

	err := binance.Init("TRY")
	if err != nil {
		// TODO: log
		os.Exit(1)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestLogger("prices"))
	r.Use(middleware.Recoverer)
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

	logger := middleware.GetLogger(r)

	exchange := strings.ToLower(chi.URLParam(r, "exchange"))
	symbol := strings.ToLower(chi.URLParam(r, "symbol"))

	start, err := strconv.ParseInt(r.URL.Query().Get("start"), 10, 64)
	if err != nil {
		start = 0
	}

	maxSize, err := strconv.ParseInt(r.URL.Query().Get("maxSize"), 10, 64)
	if err != nil {
		maxSize = 0
	}

	if exchange == "binance" {
		if !binance.CheckSymbol(symbol) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		hist, err := binance.GetSymbolHistory(symbol, start)
		if err != nil {
			logger.Error(errors.WithStack(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		hist.Prices = util.Downsample(hist.Prices, int(maxSize))
		hist.Volumes = util.Downsample(hist.Volumes, int(maxSize))

		bdy, err := json.Marshal(hist)
		if err != nil {
			logger.Error(errors.WithStack(err))
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

	logger := middleware.GetLogger(r)

	symbol := strings.ToLower(chi.URLParam(r, "symbol"))
	exchange := strings.ToLower(chi.URLParam(r, "exchange"))

	if exchange == "binance" {
		if !binance.CheckSymbol(symbol) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			logger.Error(errors.WithStack(err))
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
				logger.Error(errors.WithStack(err))
				sub.Unsubscribe()
				return
			}
		}
	}

	w.WriteHeader(http.StatusNotFound)
	return
}
