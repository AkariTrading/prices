package binance

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/akaritrading/prices/pkg/client"
)

var (
	ErrorSymbolNotFound  = errors.New("ErrorSymbolNotFound")
	ErrorUnknownExchange = errors.New("ErrorUnknownExchange")
)

const (
	OPENTIME = iota // uint
	OPEN            // string
	HIGH            // string
	LOW             // string
	CLOSE           // string
	VOLUME          // string
)

const (
	SymbolSaveInitialTimestamp = 1598804620000 // make 0 for production
)

type symbolFetchJob struct {
	Symbol string
	Save   *client.HistoryPosition
}

func fetchKlines(symbol string, save *client.HistoryPosition) ([]client.Candle, error) {

	if !CheckSymbol(symbol) {
		return nil, ErrorSymbolNotFound
	}

	query := url.Values{}
	query.Set("symbol", strings.ToUpper(symbol))
	query.Set("interval", "1m")
	query.Set("startTime", strconv.FormatInt(save.End, 10))
	query.Set("limit", strconv.FormatInt(1000, 10))

	url, _ := url.Parse("https://api.binance.com/api/v3/klines")
	url.RawQuery = query.Encode()

	res, err := requestClient.Do(&http.Request{
		Method: "GET",
		URL:    url,
	})

	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, ErrorUnknownExchange
	}

	var data [][]interface{}
	if err = json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}

	var candles []client.Candle

	for _, candle := range data {

		if save.Start == 0 {
			save.Start = int64(candle[OPENTIME].(float64))
		}

		high, _ := strconv.ParseFloat(candle[HIGH].(string), 64)
		low, _ := strconv.ParseFloat(candle[LOW].(string), 64)
		vol, _ := strconv.ParseFloat(candle[VOLUME].(string), 64)

		candles = append(candles, client.Candle{Price: float64((high + low) / 2), Volume: vol})
	}

	if len(data) > 0 {
		lastCandle := data[len(data)-1]
		lock := symbolSavesLock[symbol]
		lock.Lock()
		save.End = int64(lastCandle[OPENTIME].(float64)) + (60 * 1000) // skip one minute
		defer lock.Unlock()
	}

	return candles, nil
}

func fetchAndSaveAll() {

	jobs := make(chan symbolFetchJob, len(symbolSaves))
	workersCount := 10
	var wg sync.WaitGroup
	wg.Add(workersCount)

	for w := 0; w < workersCount; w++ {
		go fetchAndSave(jobs, &wg)
	}

	for symbol, save := range symbolSaves {
		jobs <- symbolFetchJob{Save: save, Symbol: symbol}
	}

	close(jobs)
	wg.Wait()

	fmt.Println("finished all")
}

// TODO: handle error better
func fetchAndSave(jobs chan symbolFetchJob, wg *sync.WaitGroup) {

	defer wg.Done()
	for job := range jobs {

		now := (time.Now().Unix() - (5 * 60)) * 1000 // roll back 5 minutes
		var points []client.Candle

		for job.Save.End <= now {
			candles, err := fetchKlines(job.Symbol, job.Save)
			if err == nil {
				points = append(points, candles...)
			} else if err == ErrorUnknownExchange {
				log.Fatal(err)
				return
			} else {
				log.Fatal(err)
				// TODO: log
			}
		}

		f, err := os.OpenFile(fmt.Sprintf("/priceData/binance/prices/%s", job.Symbol), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		lock := symbolSavesLock[job.Symbol]
		lock.Lock()

		client.WriteCandles(f, points)

		for _, p := range points {
			binary.Write(f, binary.LittleEndian, p)
		}
		defer lock.Unlock()
		// fmt.Println("finished ", job.Symbol)
	}
}

func GetSymbolHistory(symbol string, start int64) (*client.History, error) {

	lock := symbolSavesLock[symbol]
	lock.Lock()
	defer lock.Unlock()

	f, err := os.Open(fmt.Sprintf("/priceData/binance/prices/%s", symbol))
	if err != nil {
		// handle better
		return nil, err
	}
	defer f.Close()

	return client.HistoryWindow(f, symbolSaves[symbol], start, 0)
}
