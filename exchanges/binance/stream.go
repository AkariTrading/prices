// {
//   "e": "aggTrade",  // Event type
//   "E": 123456789,   // Event time
//   "s": "BNBBTC",    // Symbol
//   "a": 12345,       // Aggregate trade ID
//   "p": "0.001",     // Price
//   "q": "100",       // Quantity
//   "f": 100,         // First trade ID
//   "l": 105,         // Last trade ID
//   "T": 123456785,   // Trade time
//   "m": true,        // Is the buyer the market maker?
//   "M": true         // Ignore
// }

package binance

import (
	"fmt"
	"sync"
	"time"

	"github.com/akaritrading/libs/stream"
	"github.com/akaritrading/libs/util"
	"github.com/gorilla/websocket"
)

type aggTradeRaw struct {
	Symbol      string `json:"s"`
	PriceStr    string `json:"p"`
	QuantityStr string `json:"q"`
}

var streamMap sync.Map

// WEBSOCKETHOST -url for binance websocket api
var binanceWSHost = util.BinanceWSHost()

// PriceStream -
func PriceStream(symbol string) *stream.Stream {

	loadedStream, ok := streamMap.Load(symbol)

	if ok {
		return loadedStream.(*stream.Stream)
	}

	stream := stream.CreateStream()
	streamMap.Store(symbol, stream)
	go newConn(fmt.Sprintf("%s/%s%s", binanceWSHost, symbol, "@aggTrade"), stream)
	return stream
}

func newConn(url string, stream *stream.Stream) {

	for {

		var data aggTradeRaw

		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			goto reconnect
		}

		for {
			err = conn.ReadJSON(&data)
			if err != nil {
				goto reconnect
			}
			stream.Publish(data.PriceStr)
		}

	reconnect:
		time.Sleep(time.Second * 5)
		newConn(url, stream)
	}

}
