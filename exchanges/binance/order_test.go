package binance

import "testing"

func TestSign(t *testing.T) {

	key := "NhqPtmdSJYdKjVHjA7PZj4Mge3R5YNiP1e3UZjInClVN65XAbvqqM6A7H5fATj0j"
	data := "symbol=LTCBTC&side=BUY&type=LIMIT&timeInForce=GTC&quantity=1&price=0.1&recvWindow=5000&timestamp=1499827319559"

	signature, err := sign(key, data)

	if err != nil {
		t.Fatal("sign returned error")
	}

	actual := "c8db56825ae71d6d79447849e617115f4a920fa2acdcab2b053c4b2838bd6b71"

	if signature != actual {
		t.Fatal("bad signature")
	}
}
