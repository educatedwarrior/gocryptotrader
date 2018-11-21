package gemini

import (
	"errors"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// SetDefaults sets package defaults for gemini exchange
func (g *Gemini) SetDefaults() {
	g.Name = "Gemini"
	g.Enabled = false
	g.Verbose = false
	g.APIWithdrawPermissions = exchange.AutoWithdrawCryptoWithAPIPermission | exchange.AutoWithdrawCryptoWithSetup | exchange.WithdrawFiatViaWebsiteOnly
	g.RequestCurrencyPairFormat.Delimiter = ""
	g.RequestCurrencyPairFormat.Uppercase = true
	g.ConfigCurrencyPairFormat.Delimiter = ""
	g.ConfigCurrencyPairFormat.Uppercase = true
	g.AssetTypes = []string{ticker.Spot}
	g.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			AutoPairUpdates:    true,
			RESTTickerBatching: false,
			REST:               true,
			Websocket:          false,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}
	g.Requester = request.New(g.Name,
		request.NewRateLimit(time.Minute, geminiAuthRate),
		request.NewRateLimit(time.Minute, geminiUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	g.API.Endpoints.URLDefault = geminiAPIURL
	g.API.Endpoints.URL = g.API.Endpoints.URLDefault
}

// Setup sets exchange configuration parameters
func (g *Gemini) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		g.SetEnabled(false)
	} else {
		err := g.SetupDefaults(exch)
		if err != nil {
			log.Fatal(err)
		}
		if exch.UseSandbox {
			g.API.Endpoints.URL = geminiSandboxAPIURL
		}
	}
}

// Start starts the Gemini go routine
func (g *Gemini) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		g.Run()
		wg.Done()
	}()
}

// Run implements the Gemini wrapper
func (g *Gemini) Run() {
	if g.Verbose {
		log.Printf("%s %d currencies enabled: %s.\n", g.GetName(), len(g.EnabledPairs), g.EnabledPairs)
	}

	exchangeProducts, err := g.GetSymbols()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", g.GetName())
	} else {
		err = g.UpdateCurrencies(exchangeProducts, false, false)
		if err != nil {
			log.Printf("%s Failed to update available currencies.\n", g.GetName())
		}
	}
}

// GetExchangeAccountInfo Retrieves balances for all enabled currencies for the
// Gemini exchange
func (g *Gemini) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = g.GetName()
	accountBalance, err := g.GetBalances()
	if err != nil {
		return response, err
	}
	for i := 0; i < len(accountBalance); i++ {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = accountBalance[i].Currency
		exchangeCurrency.TotalValue = accountBalance[i].Amount
		exchangeCurrency.Hold = accountBalance[i].Available
		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	return response, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (g *Gemini) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := g.GetTicker(p.Pair().String())
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice.Pair = p
	tickerPrice.Ask = tick.Ask
	tickerPrice.Bid = tick.Bid
	tickerPrice.Last = tick.Last
	tickerPrice.Volume = tick.Volume.USD
	ticker.ProcessTicker(g.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(g.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (g *Gemini) FetchTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(g.GetName(), p, assetType)
	if err != nil {
		return g.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (g *Gemini) FetchOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(g.GetName(), p, assetType)
	if err != nil {
		return g.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (g *Gemini) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := g.GetOrderbook(p.Pair().String(), url.Values{})
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: orderbookNew.Bids[x].Amount, Price: orderbookNew.Bids[x].Price})
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: orderbookNew.Asks[x].Amount, Price: orderbookNew.Asks[x].Price})
	}

	orderbook.ProcessOrderbook(g.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(g.Name, p, assetType)
}

// GetExchangeFundTransferHistory returns funding history, deposits and
// withdrawals
func (g *Gemini) GetExchangeFundTransferHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, errors.New("not supported on exchange")
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (g *Gemini) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}

// SubmitExchangeOrder submits a new order
func (g *Gemini) SubmitExchangeOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// ModifyExchangeOrder will allow of changing orderbook placement and limit to
// market conversion
func (g *Gemini) ModifyExchangeOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// CancelExchangeOrder cancels an order by its corresponding ID number
func (g *Gemini) CancelExchangeOrder(orderID int64) error {
	return errors.New("not yet implemented")
}

// CancelAllExchangeOrders cancels all orders associated with a currency pair
func (g *Gemini) CancelAllExchangeOrders() error {
	return errors.New("not yet implemented")
}

// GetExchangeOrderInfo returns information on a current open order
func (g *Gemini) GetExchangeOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, errors.New("not yet implemented")
}

// GetExchangeDepositAddress returns a deposit address for a specified currency
func (g *Gemini) GetExchangeDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawCryptoExchangeFunds returns a withdrawal ID when a withdrawal is
// submitted
func (g *Gemini) WithdrawCryptoExchangeFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFunds returns a withdrawal ID when a
// withdrawal is submitted
func (g *Gemini) WithdrawFiatExchangeFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (g *Gemini) WithdrawFiatExchangeFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// GetWebsocket returns a pointer to the exchange websocket
func (g *Gemini) GetWebsocket() (*exchange.Websocket, error) {
	return nil, errors.New("not yet implemented")
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (g *Gemini) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return g.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (g *Gemini) GetWithdrawCapabilities() uint32 {
	return g.GetWithdrawPermissions()
}
