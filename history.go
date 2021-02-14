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

				var end int64
				sh, err := handle.OpenReadWrite(s)
				if err != nil {
					logger.Error(errors.WithStack(err))
					goto skip
				}

				end = sh.End()

				for {

					logger.Info(fmt.Sprintf("%s, starting fetch at %d", s, end))

					history, err := client.Klines(s, end, true)
					if err != nil {
						logger.Error(errors.WithStack(err))
						goto skip
					}

					if len(history.Candles) == 0 {
						break
					}

					end = history.End

					appendStart := sh.End()
					if appendStart == 0 {
						appendStart = history.Start
					}

					if err := sh.Append(appendStart, history.Candles); err != nil {
						logger.Error(errors.WithStack(err))
					}
				}

			skip:

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
