package pricesclient

import (
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/akaritrading/libs/util"
	"github.com/pkg/errors"
)

var client = http.Client{
	Timeout: time.Second * 10,
}

type MemoryMegabytes int

type Client struct {
	Host                 string
	Exchange             string
	OrderbookRefreshRate time.Duration

	orderbookPriceMap sync.Map
	streamMap         sync.Map
	streamPriceMap    sync.Map
}

func (c *Client) ToSymbol(symbolA string, symbolB string) string {
	if c.Exchange == "binance" {
		return symbolA + symbolB
	}

	return symbolA + symbolB
}

func getRequest(url string) ([]byte, error) {

	res, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return nil, util.ErrorSymbolNotFound
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("bad request or serviece error")
	}

	return ioutil.ReadAll(res.Body)
}

// // DO NOT TOUCH
// type Candle struct {
// 	Open   float64
// 	High   float64
// 	Low    float64
// 	Close  float64
// 	Volume float64
// }

// type HistoryPosition struct {
// 	Start int64
// 	End   int64
// }

// type HistoryPositions map[string]*HistoryPosition

// var historyPositions HistoryPositions
// var historyPositionLock sync.Mutex

// var symbolLocks map[string]*sync.Mutex
// var symbolLock sync.Mutex

// type History struct {
// 	HistoryPosition
// 	Candles []Candle
// }

// type HistoryResponse struct {
// 	HistoryPosition
// 	Candles [][]float64
// }

// func (h *HistoryResponse) ToHistory() *History {

// 	candles := make([]Candle, 0, len(h.Candles)/5)

// 	for _, arr := range h.Candles {

// 		candles = append(candles, Candle{
// 			Open:   arr[0],
// 			High:   arr[1],
// 			Low:    arr[2],
// 			Close:  arr[3],
// 			Volume: arr[4],
// 		})
// 	}

// 	return &History{
// 		HistoryPosition: HistoryPosition{
// 			Start: h.Start,
// 			End:   h.End,
// 		},
// 		Candles: candles,
// 	}
// }

// func (h *History) ToResponse() *HistoryResponse {

// 	ret := &HistoryResponse{
// 		HistoryPosition: HistoryPosition{
// 			Start: h.Start,
// 			End:   h.End,
// 		},
// 	}

// 	for _, c := range h.Candles {
// 		ret.Candles = append(ret.Candles, []float64{c.Open, c.High, c.Low, c.Close, c.Volume})
// 	}

// 	return ret

// }

// func (h *History) HighLowClose() []float64 {

// 	ret := make([]float64, 0, len(h.Candles))

// 	for _, c := range h.Candles {
// 		ret = append(ret, (c.High+c.Low+c.Close)/3.0)
// 	}

// 	return ret
// }

// func (h *History) Volumes() []float64 {

// 	ret := make([]float64, 0, len(h.Candles))

// 	for _, c := range h.Candles {
// 		ret = append(ret, c.Volume)
// 	}

// 	return ret
// }

// func (h *History) Downsample(maxSize int) []Candle {

// 	if maxSize == 0 {
// 		return h.Candles
// 	}

// 	if len(h.Candles) <= maxSize {
// 		return h.Candles
// 	}

// 	sampleRate := float64(len(h.Candles)) / float64(maxSize)

// 	ret := make([]Candle, 0, maxSize)

// 	for i := 0; i < maxSize; i++ {
// 		ret = append(ret, h.Candles[int(float64(i)*sampleRate)])
// 	}

// 	return ret
// }
