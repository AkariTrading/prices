package binance

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/akaritrading/libs/util"
	"github.com/akaritrading/prices/pkg/pricesclient"
)

const (
	OPENTIME = iota // uint
	OPEN            // string
	HIGH            // string
	LOW             // string
	CLOSE           // string
	VOLUME          // string
)

var (
	SymbolSaveInitialTimestamp = time.Now().Add(-(time.Hour * 24 * 7)).Unix() * 1000 // make 0 for production
)

type symbolFetchJob struct {
	Symbol string
	Save   *pricesclient.HistoryPosition
}

func fetchKlines(symbol string, pos *pricesclient.HistoryPosition) ([]pricesclient.Candle, error) {

	if !CheckSymbol(symbol) {
		return nil, util.ErrorSymbolNotFound
	}

	query := url.Values{}
	query.Set("symbol", strings.ToUpper(symbol))
	query.Set("interval", "1m")
	query.Set("startTime", strconv.FormatInt(pos.End, 10))
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
		fmt.Println(res.StatusCode)
		return nil, util.ErrorUnkown
	}

	var data [][]interface{}
	if err = json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}

	var candles []pricesclient.Candle

	for _, candle := range data {

		if pos.Start == 0 {
			pos.Start = int64(candle[OPENTIME].(float64))
			pos.End = int64(candle[OPENTIME].(float64))
		}

		high, _ := strconv.ParseFloat(candle[HIGH].(string), 64)
		low, _ := strconv.ParseFloat(candle[LOW].(string), 64)
		open, _ := strconv.ParseFloat(candle[OPEN].(string), 64)
		close, _ := strconv.ParseFloat(candle[CLOSE].(string), 64)
		vol, _ := strconv.ParseFloat(candle[VOLUME].(string), 64)

		candles = append(candles, pricesclient.Candle{
			High:   high,
			Low:    low,
			Open:   open,
			Close:  close,
			Volume: vol})
	}

	return candles, nil
}

func fetchAndSaveAll() {

	jobs := make(chan symbolFetchJob, len(symbolHistoryPositions))
	workersCount := 10
	var wg sync.WaitGroup
	wg.Add(workersCount)

	for w := 0; w < workersCount; w++ {
		go fetchWorker(jobs, &wg)
	}

	for symbol, save := range symbolHistoryPositions {
		jobs <- symbolFetchJob{Save: save, Symbol: symbol}
	}

	close(jobs)
	wg.Wait()
}

// TODO: handle error better
func fetchWorker(jobs chan symbolFetchJob, wg *sync.WaitGroup) {

	defer wg.Done()
	for job := range jobs {
		lock := symbolHistoryLocks[job.Symbol]
		lock.Lock()

		now := (time.Now().Unix() - (5 * 60)) * 1000 // roll back 5 minutes
		var points []pricesclient.Candle

		for atomic.LoadInt64(&job.Save.End) <= now {
			candles, err := fetchKlines(job.Symbol, job.Save)
			if err != nil {
				fmt.Println(err)
				break
			}
			atomic.AddInt64(&job.Save.End, int64(len(candles)*1000*60))
			points = append(points, candles...)
		}

		if len(points) > 0 {

			f, err := os.OpenFile(fmt.Sprintf("/priceData/binance/prices/%s", job.Symbol), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			defer f.Sync()

			pricesclient.WriteCandles(f, points)

		}
		lock.Unlock()
	}
}

func GetSymbolHistory(symbol string, start int64) (*pricesclient.History, error) {

	// defer util.TimeTrack(time.Now(), "GetSymbolHistory")

	lock := symbolHistoryLocks[symbol]
	lock.RLock()
	defer lock.RUnlock()

	f, err := os.Open(fmt.Sprintf("/priceData/binance/prices/%s", symbol))
	if err != nil {
		// handle better
		return nil, err
	}
	defer f.Close()

	return pricesclient.ReadHistoryWindow(f, symbolHistoryPositions[symbol], start, 0)
}
