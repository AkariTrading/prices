package client

import (
	"fmt"
	"strconv"
	"time"

	"github.com/akaritrading/libs/stream"
	"github.com/gorilla/websocket"
)

// PriceStream -
func (c *PriceClient) PriceStream(symbol string) (*stream.Stream, error) {

	symbolStream, ok := c.streamMap.Load(symbol)

	if ok {
		return symbolStream.(*stream.Stream), nil
	}

	return c.initStream(symbol)
}

// StreamRecentPrice - does not check if symbol is valid. PriceStream must be called first to initiate the stream.
func (c *PriceClient) StreamRecentPrice(symbol string) float64 {

	if p, ok := c.streamPriceMap.Load(symbol); ok {
		return p.(float64)
	}

	return 0
}

func (c *PriceClient) initStream(symbol string) (*stream.Stream, error) {

	price, err := c.OrderbookPrice(symbol)
	if err != nil {
		return nil, err
	}

	c.streamPriceMap.Store(symbol, (price.Buy+price.Sell)/2)

	stream := stream.CreateStream()
	c.streamMap.Store(symbol, stream)
	go c.newConn(symbol, stream)
	return stream, nil
}

func (c *PriceClient) newConn(symbol string, stream *stream.Stream) {

	var priceStr string

	url := fmt.Sprintf("ws://%s/%s/priceStream/%s", c.Host, c.Exchange, symbol)

	for {

		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			fmt.Println(err)
			goto reconnect
		}

		for {
			err = conn.ReadJSON(&priceStr)
			if err != nil {
				fmt.Println(err)
				conn.Close()
				goto reconnect
			}
			if f, err := strconv.ParseFloat(priceStr, 64); err == nil {
				c.streamPriceMap.Store(symbol, f)
				stream.Publish(f)
			} else {
				// TODO: report parsing error
			}
		}

	reconnect:
		time.Sleep(time.Second * 5)
		// log reconnecting
	}
}
