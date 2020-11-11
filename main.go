package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akaritrading/libs/exchange/binance"
	"github.com/akaritrading/libs/exchange/candlefs"
	"github.com/akaritrading/libs/flag"
	"github.com/akaritrading/libs/middleware"
	"github.com/akaritrading/libs/util"
	"github.com/go-chi/chi"
	"github.com/pkg/errors"
)

var binanceClient *binance.BinanceClient
var binanceCandlefs *candlefs.CandleFS

func main() {

	binanceClient = &binance.BinanceClient{}
	err := binanceClient.FetchSymbols("TRY")
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll("/candles/binance/", 0644)
	if err != nil {
		log.Fatal(err)
	}

	binanceCandlefs = candlefs.New("/candles/binance/")

	stopJob := make(chan int)
	onExit(stopJob)

	StartHistoryFetch(binanceCandlefs, binanceClient, stopJob)

	r := chi.NewRouter()
	r.Use(middleware.RequestContext("prices", nil))
	r.Use(middleware.Recoverer)

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

	start, err := util.StrToInt(r.URL.Query().Get("start"))
	if err != nil {
		start = 0
	}

	end, err := util.StrToInt(r.URL.Query().Get("end"))
	if err != nil {
		end = time.Now().Unix() * 1000
	}

	maxSize, err := util.StrToInt(r.URL.Query().Get("maxSize"))
	if err != nil {
		maxSize = 0
	}

	if exchange == "binance" {

		if err := binanceClient.CheckSymbol(symbol); err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		symbolHandle, err := binanceCandlefs.Open(symbol)
		if err != nil {
			logger.Error(errors.WithStack(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer symbolHandle.Close()

		hist, err := symbolHandle.Read(start, end)
		if err != nil {
			logger.Error(errors.WithStack(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		hist.Candles = hist.Downsample(int(maxSize))
		bdy, err := json.Marshal(hist.ToFlat())
		if err != nil {
			logger.Error(errors.WithStack(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(bdy)
	}
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
