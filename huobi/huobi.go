package huobi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"regexp"
	"log"
)

type Symbol struct {
	BaseCurrency    Currency `json:"base-currency"`
	QuoteCurrency   Currency `json:"quote-currency"`
	PricePrecision  uint     `json:"price-precision"`
	AmountPrecision uint     `json:"amount-precision"`
	SymbolPartition string   `json:"symbol-partition"`
}

type PriceAmount []float64
type MarketDepth struct {
	Bids      []PriceAmount
	Asks      []PriceAmount
	Ts        uint `json:"ts"`
	Version   uint `json:"version"`
	SymbolStr string
}
type ExchangeRate struct {
	From Currency
	To   Currency
	Rate float64
	Max  float64
}

type Currency string
type Chain []Currency

const (
	UA      = "Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/63.0.3239.132 Safari/537.36"
	BaseUrl = "http://api.huobipro.com"
)

func (s *Symbol) Members() []Currency {
	return []Currency{s.BaseCurrency, s.QuoteCurrency}
}

func (s *Symbol) Has(c Currency) bool {
	for _, v := range s.Members() {
		if v == c {
			return true
		}
	}
	return false
}

func (s *Symbol) Symbol() string {
	return string(s.BaseCurrency + s.QuoteCurrency)
}

func (c Chain) Pairs() [][]Currency {
	return [][]Currency{[]Currency{c[0], c[1]}, []Currency{c[1], c[2]}, []Currency{c[2], c[0]}}
}

func (c Chain) Try(symbols []Symbol) (float64, float64){
	//symbols, _ := Symbols()
	var rates []ExchangeRate
	for _, p := range c.Pairs() {
		depth, _ := Depth(getSymbolStr(p, symbols), 0)
		rates = append(rates, GetExchangeRate(p[0], p[1], depth))
	}
	min := math.Min(rates[0].Max, rates[1].Max*rates[0].Rate)
	min = math.Min(min, rates[2].Max*rates[1].Rate*rates[0].Rate)
	var rate float64
	rate = 1 / (rates[0].Rate * rates[1].Rate * rates[2].Rate)
	rate -= 0.002
	return min, rate
}

// Symbols 获取交易对
func Symbols() (symbols []Symbol, err error) {

	type S struct {
		Status string   `json:"status"`
		Data   []Symbol `json:"data"`
	}

	// 接口URL
	api := "/v1/common/symbols"
	url := fmt.Sprintf("%s%s", BaseUrl, api)

	// 初始化
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)

	// 设置UA
	req.Header.Set("User-Agent", UA)

	// 访问
	resp, err := client.Do(req)
	if err != nil {
		return symbols, err
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("access failed")
		return symbols, err
	}

	// 获取内容
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	s := new(S)
	err = json.Unmarshal(bodyBytes, s)

	if err != nil {
		return symbols, err
	}

	if s.Status == "error" {
		err = fmt.Errorf("exchange server error")
		return symbols, err
	}

	return s.Data, err
}

// GetTriangleChains 获取三角兑换关系
func GetTriangleChains(symbols []Symbol, start Currency) (chains []Chain) {
	var pairs [][]Symbol
	for _, s := range symbols {
		for _, s1 := range symbols {
			if s == s1 {
				continue
			}
			if !s.Has(start) || !s1.Has(start) {
				continue
			}
			p := []Symbol{s, s1}
			pairs = append(pairs, p)
		}
	}
	for _, p := range pairs {
		var members []Currency
		for _, v := range append(p[0].Members(), p[1].Members()...) {
			if v != start {
				members = append(members, v)
			}
		}
		if len(members) != 2 {
			continue
		}
		for _, s := range symbols {
			if s.Has(members[0]) && s.Has(members[1]) {
				chains = append(chains, append(Chain{start}, members...))
				chains = append(chains, append(Chain{start}, members[1], members[0]))
			}
		}
	}
	return
}

// Depth 获取市场深度
func Depth(symbolStr string, depthType int) (md MarketDepth, err error) {

	type S struct {
		Status string      `json:"status"`
		Ch     string      `json:"ch"`
		Ts     uint        `json:"ts"`
		Tick   MarketDepth `json:"tick"`
	}

	// 接口URL
	api := "/market/depth"
	url := fmt.Sprintf("%s%s?symbol=%s&type=step%d", BaseUrl, api, symbolStr, depthType)

	// 初始化
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)

	// 设置UA
	req.Header.Set("User-Agent", UA)

	// 访问
	resp, err := client.Do(req)
	if err != nil {
		return
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("access failed")
		return
	}

	// 获取内容
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	s := new(S)
	err = json.Unmarshal(bodyBytes, s)

	if err != nil {
		return
	}

	if s.Status == "error" {
		err = fmt.Errorf("exchange server error")
		return
	}

	md = s.Tick
	md.SymbolStr = symbolStr
	return

}

// getSymbolStr 获取symbol字符串
func getSymbolStr(pair []Currency, symbols []Symbol) (symbolStr string) {
	for _, s := range symbols {
		if s.Has(pair[0]) && s.Has(pair[1]) {
			return s.Symbol()
		}
	}
	return
}

// GetExchangeRate 获取兑换比例
func GetExchangeRate(from, to Currency, depth MarketDepth) (er ExchangeRate) {
	er.From = from // usdt
	er.To = to     // btc

	var data []PriceAmount
	pattern := "^" + string(from)
	re, _ := regexp.Compile(pattern)
	// btcusdt
	if re.MatchString(depth.SymbolStr) {
		data = depth.Bids
	} else {
		data = depth.Asks
	}
	var amount, sum float64
	max := 1
	for _, v := range data {
		if max == 0 {
			break
		}
		sum += v[0] * v[1]
		amount += v[1]
		max -= 1
	}

	if re.MatchString(depth.SymbolStr) {
		er.Rate = amount / sum
		er.Max = amount
	} else {
		er.Rate = sum / amount
		er.Max = sum
	}

	return
}

// Run
func Run() (err error){
	symbols, err := Symbols()
	var start Currency = "usdt"
	chains := GetTriangleChains(symbols, start)

	for {
			for _, c := range chains {
				max, rate:= c.Try(symbols)
				log.Println(c, max, rate)
				//log.Print(c.Try())
			}
	}
	return
}

