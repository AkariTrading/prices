package binance

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/akaritrading/libs/util"
)

type Account struct {
	MakerFee float64
	TakerFee float64
	// CanTrade        bool
	// CanWithdraw     bool
	// CanDeposit      bool
	Balances map[string]float64
}

type Balance struct {
	Asset  string
	Free   float64
	Locked float64
}

func (u *UserClient) Account() (*Account, error) {

	params := make(map[string]string)
	params["timestamp"] = strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	params["recvWindow"] = "5000"

	req, err := u.newRequest("GET", binanceAccountURL, params)
	if err != nil {
		return nil, err
	}

	res, err := httpclient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	rawAccount := struct {
		MakerCommission  int64 `json:"makerCommission"`
		TakerCommission  int64 `json:"takerCommission"`
		BuyerCommission  int64 `json:"buyerCommission"`
		SellerCommission int64 `json:"sellerCommission"`
		CanTrade         bool  `json:"canTrade"`
		CanWithdraw      bool  `json:"canWithdraw"`
		CanDeposit       bool  `json:"canDeposit"`
		Balances         []struct {
			Asset  string `json:"asset"`
			Free   string `json:"free"`
			Locked string `json:"locked"`
		}
	}{}

	json.NewDecoder(res.Body).Decode(&rawAccount)
	if err != nil {
		return nil, err
	}

	acc := &Account{
		MakerFee: float64(rawAccount.MakerCommission) / 10000.0,
		TakerFee: float64(rawAccount.TakerCommission) / 10000.0,
		Balances: make(map[string]float64),
	}

	for _, b := range rawAccount.Balances {
		f, err := util.StrToFloat64(b.Free)
		if err != nil {
			return nil, err
		}

		if f == 0 {
			continue
		}

		acc.Balances[strings.ToLower(b.Asset)] = f
	}

	return acc, nil
}
