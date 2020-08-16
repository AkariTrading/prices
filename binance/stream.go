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

	"github.com/gorilla/websocket"
)

type aggTradeRaw struct {
	Symbol      string `json:"s"`
	PriceStr    string `json:"p"`
	QuantityStr string `json:"q"`
}

// Stream -
type Stream struct {
	stopCh    chan struct{}
	publishCh chan string
	subCh     chan chan interface{}
	unsubCh   chan chan interface{}
}

var streamMap sync.Map

// PriceStream -
func PriceStream(symbol string) *Stream {

	loadedStream, ok := streamMap.Load(symbol)

	if ok {
		fmt.Println("FROM CACHE")
		return loadedStream.(*Stream)
	}

	stream := createStrem()
	streamMap.Store(symbol, stream)
	go stream.start()
	go newConn(fmt.Sprintf("%s/%s%s", websockethost, symbol, "@aggTrade"), stream)
	return stream
}

func newConn(url string, stream *Stream) {

	defer func() {
		time.Sleep(time.Second * 5)
		newConn(url, stream)
	}()

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return
	}

	var data aggTradeRaw
	for {
		err = conn.ReadJSON(&data)
		if err != nil {
			return
		}
		stream.publish(data.PriceStr)
	}
}

func createStrem() *Stream {
	return &Stream{
		stopCh:    make(chan struct{}),
		publishCh: make(chan string),
		subCh:     make(chan chan interface{}, 1),
		unsubCh:   make(chan chan interface{}, 1),
	}
}

func (b *Stream) start() {
	subs := map[chan interface{}]struct{}{}
	for {
		select {
		case <-b.stopCh:
			return
		case msgCh := <-b.subCh:
			subs[msgCh] = struct{}{}
		case msgCh := <-b.unsubCh:
			delete(subs, msgCh)
		case msg := <-b.publishCh:
			for msgCh := range subs {
				select {
				case msgCh <- msg:
				default:
				}
			}
		}
	}
}

// Subscribe -
func (b *Stream) Subscribe() chan interface{} {
	msgCh := make(chan interface{}, 5)
	b.subCh <- msgCh
	return msgCh
}

// Unsubscribe -
func (b *Stream) Unsubscribe(msgCh chan interface{}) {
	b.unsubCh <- msgCh
}

func (b *Stream) stop() {
	close(b.stopCh)
}

func (b *Stream) publish(msg string) {
	b.publishCh <- msg
}
