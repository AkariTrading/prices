package main

import (
	"time"

	"github.com/akaritrading/libs/exchange"
	"github.com/akaritrading/libs/exchange/candlefs"
	"github.com/akaritrading/libs/log"
	"github.com/pkg/errors"
)

func StartHistoryFetch(handle *candlefs.CandleFS, client exchange.Exchange, stop chan int) {

	logger := log.New("prices", "")

	go func() {

		symbols := client.Symbols()

		for {

			for _, s := range symbols {

				var candles []exchange.Candle
				var trueStart int64
				var start int64

				sh, err := handle.Open(s)
				if err != nil {
					logger.Error(errors.WithStack(err))
					goto skip
				}

				start = sh.End()

				for {
					history, err := client.Klines(s, start, true)
					if err != nil {
						logger.Error(errors.WithStack(err))
						goto skip
					}

					if trueStart == 0 {
						trueStart = history.Start
					}

					if history.End > 0 {
						start = history.End
					}

					candles = append(candles, history.Candles...)

					if len(history.Candles) == 0 {
						break
					}
				}

			skip:
				if trueStart > 0 {
					if err := sh.Append(trueStart, candles); err != nil {
						logger.Error(errors.WithStack(err))
					}
				}
				if sh != nil {
					sh.Close()
				}

				select {
				case <-stop:
					return
				default:
				}
			}

			select {
			case <-stop:
				return
			case <-time.NewTimer(time.Minute * 5).C:
				logger.Info("sleeping")
			}
		}
	}()

}
