package qtrade

import (
"testing"
	"carrytrade/huobi"
)

func TestSymbols(t *testing.T) {
	symbols, err := huobi.Symbols()
	if err != nil {
		t.Fatal(err)
	}

	if len(symbols) == 0 {
		t.Fatal(symbols)
	}

	t.Log(symbols[0])

	t.Log(len(symbols))
}

func TestGetTriangleChains(t *testing.T) {
	symbols, _ := huobi.Symbols()
	var start huobi.Currency = "usdt"
	chains := huobi.GetTriangleChains(symbols, start)
	t.Log(len(chains))
	for _, v := range chains {
		t.Log(v)
	}
}

func TestDepth(t *testing.T) {
	md, err := huobi.Depth("btcusdt", 1)
	if err != nil {
		t.Log(err)
	}
	t.Log(md)
}

func TestRun(t *testing.T) {
	huobi.Run()
}