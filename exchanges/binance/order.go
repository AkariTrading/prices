package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/akaritrading/libs/util"
)

var httpclient = http.Client{
	Timeout: time.Second * 15,
}

const (
	binanceOrderURL   = "https://www.binance.com/api/v3/order"
	binanceAccountURL = "https://www.binance.com/api/v3/account"
)

type UserClient struct {
	ApiKey string
	Secret string
}

type OrderSide string

const (
	SideSell OrderSide = "SELL"
	SideBuy  OrderSide = "BUY"
)

type MarketOrder struct {
	Symbol   string
	Side     OrderSide
	Quantity float64
}

type orderResp struct {
	Price               string
	OrigQty             string
	ExecutedQty         string
	Status              string
	CummulativeQuoteQty string
}

type errorResponse struct {
	Code int
	Msg  string
}

func (u *UserClient) newRequest(method string, url string, params map[string]string) (*http.Request, error) {

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return req, err
	}

	req.Header.Add("X-MBX-APIKEY", u.ApiKey)

	q := req.URL.Query()
	for key, val := range params {
		q.Add(key, val)
	}

	sum, err := sign(u.Secret, q.Encode())
	if err != nil {
		return req, err
	}

	q.Set("signature", sum)
	req.URL.RawQuery = q.Encode()

	return req, err
}

// MarketOrder - (ex: BTCUSDT) if side is SELL, quantity is BTC
// whereas, if side is BUY, quantity is USDT
func (u *UserClient) MarketOrder(symbolStr string, side OrderSide, quantity float64) (float64, error) {

	symbol, ok := GetSymbol(symbolStr)
	if !ok {
		return 0, util.ErrorSymbolNotFound
	}

	params := make(map[string]string)
	params["symbol"] = symbolStr
	params["side"] = string(side)
	params["type"] = "MARKET"
	params["timestamp"] = strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)

	if side == SideSell {
		params["quantity"] = strconv.FormatFloat(float64(int64(quantity/symbol.LotSize))*symbol.LotSize, 'f', symbol.BaseAssetPrecision, 64)
	} else {
		params["quoteOrderQty"] = strconv.FormatFloat(quantity, 'f', symbol.QuotePrecision, 64)
	}

	req, err := u.newRequest("POST", binanceOrderURL, params)
	if err != nil {
		return 0.0, err
	}

	res, err := httpclient.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		var body errorResponse
		json.NewDecoder(res.Body).Decode(&body)
		return 0, errors.New(body.Msg)
	}

	var body orderResp
	json.NewDecoder(res.Body).Decode(&body)

	if side == SideSell {
		return strconv.ParseFloat(body.CummulativeQuoteQty, 64)
	}

	return strconv.ParseFloat(body.OrigQty, 64)
}

func sign(key string, data string) (string, error) {
	mac := hmac.New(sha256.New, []byte(key))
	_, err := mac.Write([]byte(data))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(mac.Sum(nil)), nil
}
