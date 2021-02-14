package main

import (
	"fmt"
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
				var start int64

				sh, err := handle.OpenReadWrite(s)
				if err != nil {
					logger.Error(errors.WithStack(err))
					goto skip
				}
				defer sh.Close()

				start = sh.End()

				for {

					logger.Info(fmt.Sprintf("%s, starting fetch at %d", s, start))

					history, err := client.Klines(s, start, true)
					if err != nil {
						logger.Error(errors.WithStack(err))
						goto skip
					}

					start = history.End
					candles = append(candles, history.Candles...)

					if len(history.Candles) == 0 {
						break
					}
				}

			skip:
				if len(candles) > 0 {
					if err := sh.Append(sh.End(), candles); err != nil {
						logger.Error(errors.WithStack(err))
					}
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
