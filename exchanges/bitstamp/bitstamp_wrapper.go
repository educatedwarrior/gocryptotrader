package bitstamp

import (
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// SetDefaults sets default for Bitstamp
func (b *Bitstamp) SetDefaults() {
	b.Name = "Bitstamp"
	b.Enabled = false
	b.Verbose = false
	b.APIWithdrawPermissions = exchange.AutoWithdrawCrypto | exchange.AutoWithdrawFiat
	b.RequestCurrencyPairFormat.Delimiter = ""
	b.RequestCurrencyPairFormat.Uppercase = true
	b.ConfigCurrencyPairFormat.Delimiter = ""
	b.ConfigCurrencyPairFormat.Uppercase = true
	b.AssetTypes = []string{ticker.Spot}
	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			AutoPairUpdates:    true,
			RESTTickerBatching: false,
			REST:               true,
			Websocket:          true,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}
	b.Requester = request.New(b.Name,
		request.NewRateLimit(time.Minute*10, bitstampAuthRate),
		request.NewRateLimit(time.Minute*10, bitstampUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	b.API.Endpoints.URLDefault = bitstampAPIURL
	b.API.Endpoints.URL = b.API.Endpoints.URLDefault
	b.WebsocketInit()
	b.API.CredentialsValidator.RequiresClientID = true
}

// Setup sets configuration values to bitstamp
func (b *Bitstamp) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		b.SetEnabled(false)
	} else {
		err := b.SetupDefaults(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = b.WebsocketSetup(b.WsConnect,
			exch.Name,
			exch.Features.Enabled.Websocket,
			BitstampPusherKey,
			exch.API.Endpoints.WebsocketURL)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// Start starts the Bitstamp go routine
func (b *Bitstamp) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the Bitstamp wrapper
func (b *Bitstamp) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s.", b.GetName(), common.IsEnabled(b.Websocket.IsEnabled()))
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	pairs, err := b.GetTradingPairs()
	if err != nil {
		log.Printf("%s failed to get trading pairs. Err: %s", b.Name, err)
	} else {
		var currencies []string
		for x := range pairs {
			if pairs[x].Trading != "Enabled" {
				continue
			}
			pair := strings.Split(pairs[x].Name, "/")
			currencies = append(currencies, pair[0]+pair[1])
		}
		err = b.UpdateCurrencies(currencies, false, false)
		if err != nil {
			log.Printf("%s Failed to update available currencies.\n", b.Name)
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bitstamp) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := b.GetTicker(p.Pair().String(), false)
	if err != nil {
		return tickerPrice, err

	}
	tickerPrice.Pair = p
	tickerPrice.Ask = tick.Ask
	tickerPrice.Bid = tick.Bid
	tickerPrice.Low = tick.Low
	tickerPrice.Last = tick.Last
	tickerPrice.Volume = tick.Volume
	tickerPrice.High = tick.High
	ticker.ProcessTicker(b.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(b.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (b *Bitstamp) FetchTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tick, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Bitstamp) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return b.GetFee(feeBuilder)

}

// FetchOrderbook returns the orderbook for a currency pair
func (b *Bitstamp) FetchOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitstamp) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := b.GetOrderbook(p.Pair().String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data.Amount, Price: data.Price})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data.Amount, Price: data.Price})
	}

	orderbook.ProcessOrderbook(b.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(b.Name, p, assetType)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// Bitstamp exchange
func (b *Bitstamp) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = b.GetName()
	accountBalance, err := b.GetBalance()
	if err != nil {
		return response, err
	}

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "BTC",
		TotalValue:   accountBalance.BTCAvailable,
		Hold:         accountBalance.BTCReserved,
	})

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "XRP",
		TotalValue:   accountBalance.XRPAvailable,
		Hold:         accountBalance.XRPReserved,
	})

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "USD",
		TotalValue:   accountBalance.USDAvailable,
		Hold:         accountBalance.USDReserved,
	})

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "EUR",
		TotalValue:   accountBalance.EURAvailable,
		Hold:         accountBalance.EURReserved,
	})
	return response, nil
}

// GetExchangeFundTransferHistory returns funding history, deposits and
// withdrawals
func (b *Bitstamp) GetExchangeFundTransferHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, errors.New("not supported on exchange")
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Bitstamp) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}

// SubmitExchangeOrder submits a new order
func (b *Bitstamp) SubmitExchangeOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// ModifyExchangeOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bitstamp) ModifyExchangeOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// CancelExchangeOrder cancels an order by its corresponding ID number
func (b *Bitstamp) CancelExchangeOrder(orderID int64) error {
	return errors.New("not yet implemented")
}

// CancelAllExchangeOrders cancels all orders associated with a currency pair
func (b *Bitstamp) CancelAllExchangeOrders() error {
	return errors.New("not yet implemented")
}

// GetExchangeOrderInfo returns information on a current open order
func (b *Bitstamp) GetExchangeOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, errors.New("not yet implemented")
}

// GetExchangeDepositAddress returns a deposit address for a specified currency
func (b *Bitstamp) GetExchangeDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawCryptoExchangeFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bitstamp) WithdrawCryptoExchangeFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bitstamp) WithdrawFiatExchangeFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bitstamp) WithdrawFiatExchangeFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *Bitstamp) GetWebsocket() (*exchange.Websocket, error) {
	return b.Websocket, nil
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (b *Bitstamp) GetWithdrawCapabilities() uint32 {
	return b.GetWithdrawPermissions()
}
