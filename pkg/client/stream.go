package client

import (
	"fmt"
	"strconv"
	"time"

	"github.com/akaritrading/libs/stream"
	"github.com/gorilla/websocket"
)

// PriceStream -
func (c *PriceClient) PriceStream(symbol string) *stream.Stream {

	loadedStream, ok := c.streamMap.Load(symbol)

	if ok {
		fmt.Println("FROM CACHE")
		return loadedStream.(*stream.Stream)
	}

	// here:  must check if pair is valid with price service!!

	stream := stream.CreateStream()
	c.streamMap.Store(symbol, stream)
	go newConn(fmt.Sprintf("ws://%s/%s/priceStream/%s", c.Host, c.Exchange, symbol), stream)
	return stream
}

func newConn(url string, stream *stream.Stream) {

	var priceStr string

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
				stream.Publish(f)
			} else {
				// TODO: report parsing error
			}
		}

	reconnect:
		time.Sleep(time.Second * 5)
	}
}
