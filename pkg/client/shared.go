package client

import (
	"errors"
	"io/ioutil"
	"net/http"
	"time"
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
		return nil, ErrorSymbolNotFound
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("bad request or serviece error")
	}

	return ioutil.ReadAll(res.Body)
}
