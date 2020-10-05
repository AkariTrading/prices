package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/akaritrading/libs/exchange/binance"
	"github.com/akaritrading/libs/flag"
	"github.com/akaritrading/libs/middleware"
	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var binanceHistory *ExchangeHistory
var binanceClient *binance.BinanceClient

func main() {

	binanceClient = &binance.BinanceClient{}
	err := binanceClient.Init("TRY")
	if err != nil {
		log.Fatal(err)
	}

	stopJob := make(chan int)
	onExit(stopJob)

	binanceHistory, err = InitHistoryJob("binance", binanceClient, stopJob)
	if err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestContext("prices", nil))
	r.Use(middleware.Recoverer)
	r.Get("/{exchange}/priceStream/{symbol}", pricesWebsocket)

	// r.Get("/{exchange}/orderbookPrice/{symbol}", orderbookPrice)
	// r.Get("/{exchange}/symbols", symbolHandle)

	r.Route("/{exchange}/history/{symbol}", func(newRoute chi.Router) {
		// newRoute.Use(chimiddleware.Compress(5))
		newRoute.Get("/", priceHistory)
	})

	server := &http.Server{
		Addr:    flag.PricesHost(),
		Handler: r,
	}

	server.ListenAndServe()
}

func priceHistory(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	logger := middleware.GetLogger(r)

	exchange := chi.URLParam(r, "exchange")
	symbol := chi.URLParam(r, "symbol")

	start, err := strconv.ParseInt(r.URL.Query().Get("start"), 10, 64)
	if err != nil {
		start = 0
	}

	maxSize, err := strconv.ParseInt(r.URL.Query().Get("maxSize"), 10, 64)
	if err != nil {
		maxSize = 0
	}

	if exchange == "binance" {
		if err := binanceClient.CheckSymbol(symbol); err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		hist, err := binanceHistory.GetSymbolHistory(symbol, start)
		if err != nil {
			logger.Error(errors.WithStack(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		hist.Candles = hist.Downsample(int(maxSize))

		bdy, err := json.Marshal(hist.ToResponse())
		if err != nil {
			logger.Error(errors.WithStack(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(bdy)
	}
}

// func orderbookPrice(w http.ResponseWriter, r *http.Request) {

// 	symbol := strings.ToLower(chi.URLParam(r, "symbol"))
// 	exchange := strings.ToLower(chi.URLParam(r, "exchange"))

// 	if exchange == "binance" {
// 		if !binance.CheckSymbol(symbol) {
// 			w.WriteHeader(http.StatusNotFound)
// 			return
// 		}

// 		price, _ := binance.OrderBookPrice(symbol)
// 		js, _ := json.Marshal(price)
// 		w.Write(js)
// 		return
// 	}

// 	util.ErrorJSON(w, util.ErrorExchangeNotFound)
// 	w.WriteHeader(http.StatusNotFound)
// 	return
// }

func pricesWebsocket(w http.ResponseWriter, r *http.Request) {

	logger := middleware.GetLogger(r)

	symbol := chi.URLParam(r, "symbol")
	exchange := chi.URLParam(r, "exchange")

	if exchange == "binance" {
		if err := binanceClient.CheckSymbol(symbol); err != nil {
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
		defer sub.Unsubscribe()

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

func onExit(stop ...chan int) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("notify")
		for _, s := range stop {
			s <- 1
		}
		os.Exit(0)
	}()
}
