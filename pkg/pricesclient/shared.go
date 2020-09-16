package pricesclient

import (
	"io/ioutil"
	"net/http"
	"time"

	"github.com/akaritrading/libs/util"
	"github.com/pkg/errors"
)

var client = http.Client{
	Timeout: time.Second * 10,
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
