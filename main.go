package main

import (
	"net/http"
	"os"
	"path"
	"time"

	"github.com/akaritrading/libs/errutil"
	"github.com/akaritrading/libs/exchange/binance"
	"github.com/akaritrading/libs/exchange/candlefs"
	"github.com/akaritrading/libs/flag"
	"github.com/akaritrading/libs/log"
	"github.com/akaritrading/libs/middleware"
	"github.com/akaritrading/libs/util"
	"github.com/go-chi/chi"
	"github.com/pkg/errors"
)

var binanceClient *binance.BinanceClient
var binanceCandlefs *candlefs.CandleFS

func main() {

	logger := log.New("prices", "")
	binanceClient = &binance.BinanceClient{}
	err := binanceClient.FetchSymbols()
	if err != nil {
		logger.Fatal(err)
	}

	binancePath := path.Join(flag.PricesPath(), "binance")

	err = os.MkdirAll(binancePath, 0644)
	if err != nil {
		logger.Fatal(err)
	}

	binanceCandlefs = candlefs.New(binancePath)

	stopJob := make(chan int)
	util.OnExit(stopJob)
	StartHistoryFetch(binanceCandlefs, binanceClient, stopJob)

	r := chi.NewRouter()
	r.Use(middleware.RequestContext("prices", nil))
	r.Use(middleware.Recoverer)
	r.Use(middleware.JSONResponse)

	r.Route("/{exchange}/history/{symbol}", func(newRoute chi.Router) {
		// newRoute.Use(chimiddleware.Compress(5))
		newRoute.Get("/", priceHistory)
	})

	server := &http.Server{
		Addr:    flag.ServicePort("prices"),
		Handler: r,
	}

	logger.Fatal(server.ListenAndServe())
}

func priceHistory(w http.ResponseWriter, r *http.Request) {

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

	// maxSize, err := util.StrToInt(r.URL.Query().Get("maxSize"))
	// if err != nil {
	// 	maxSize = 0
	// }

	if exchange == "binance" {

		if err := binanceClient.CheckSymbol(symbol); err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		sh, err := binanceCandlefs.OpenRead(symbol)
		if err != nil {
			logger.Error(errors.WithStack(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer sh.Close()

		data, err := sh.Encode(start, end)
		if err != nil {
			logger.Error(errors.WithStack(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Write(data)
	} else {
		errutil.ErrorJSON(w, util.ErrorExchangeNotFound)
	}
}

// not needed
// func createZip(path string, exchange string) error {

// 	zipFile, err := os.Create(fmt.Sprintf("%s/%s.zip", path, exchange))
// 	if err != nil {
// 		return err
// 	}
// 	defer zipFile.Close()

// 	w := zip.NewWriter(zipFile)

// 	files, err := ioutil.ReadDir(fmt.Sprintf("%s/%s", path, exchange))
// 	if err != nil {
// 		return err
// 	}

// 	for _, f := range files {
// 		data, err := ioutil.ReadFile(fmt.Sprintf("%s/%s/%s", path, exchange, f.Name()))
// 		if err != nil {
// 			return err
// 		}

// 		fz, err := w.Create(f.Name())
// 		if err != nil {
// 			return err
// 		}

// 		_, err = fz.Write(data)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	return w.Close()
// }
