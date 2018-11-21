package btcc

import (
	"errors"
	"log"
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

// SetDefaults sets default values for the exchange
func (b *BTCC) SetDefaults() {
	b.Name = "BTCC"
	b.Enabled = false
	b.Fee = 0
	b.Verbose = false
	b.APIWithdrawPermissions = exchange.NoAPIWithdrawalMethods
	b.RequestCurrencyPairFormat.Delimiter = ""
	b.RequestCurrencyPairFormat.Uppercase = true
	b.ConfigCurrencyPairFormat.Delimiter = ""
	b.ConfigCurrencyPairFormat.Uppercase = true
	b.AssetTypes = []string{ticker.Spot}
	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			AutoPairUpdates:    true,
			RESTTickerBatching: false,
			REST:               false,
			Websocket:          true,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}
	b.Requester = request.New(b.Name,
		request.NewRateLimit(time.Second, btccAuthRate),
		request.NewRateLimit(time.Second, btccUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	b.WebsocketInit()
}

// Setup is run on startup to setup exchange with config values
func (b *BTCC) Setup(exch config.ExchangeConfig) {
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
			btccSocketioAddress,
			exch.API.Endpoints.WebsocketURL)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// Start starts the BTCC go routine
func (b *BTCC) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the BTCC wrapper
func (b *BTCC) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s.", b.GetName(), common.IsEnabled(b.Websocket.IsEnabled()))
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	if common.StringDataContains(b.EnabledPairs, "CNY") || common.StringDataContains(b.AvailablePairs, "CNY") || common.StringDataContains(b.BaseCurrencies, "CNY") {
		log.Println("WARNING: BTCC only supports BTCUSD now, upgrading available, enabled and base currencies to BTCUSD/USD")
		pairs := []string{"BTCUSD"}
		cfg := config.GetConfig()
		exchCfg, err := cfg.GetExchangeConfig(b.Name)
		if err != nil {
			log.Printf("%s failed to get exchange config. %s\n", b.Name, err)
			return
		}

		exchCfg.BaseCurrencies = "USD"
		exchCfg.AvailablePairs = pairs[0]
		exchCfg.EnabledPairs = pairs[0]
		b.BaseCurrencies = []string{"USD"}

		err = b.UpdateCurrencies(pairs, false, true)
		if err != nil {
			log.Printf("%s failed to update available currencies. %s\n", b.Name, err)
		}

		err = b.UpdateCurrencies(pairs, true, true)
		if err != nil {
			log.Printf("%s failed to update enabled currencies. %s\n", b.Name, err)
		}

		err = cfg.UpdateExchangeConfig(exchCfg)
		if err != nil {
			log.Printf("%s failed to update config. %s\n", b.Name, err)
			return
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *BTCC) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	// var tickerPrice ticker.Price
	// tick, err := b.GetTicker(exchange.FormatExchangeCurrency(b.GetName(), p).String())
	// if err != nil {
	// 	return tickerPrice, err
	// }
	// tickerPrice.Pair = p
	// tickerPrice.Ask = tick.AskPrice
	// tickerPrice.Bid = tick.BidPrice
	// tickerPrice.Low = tick.Low
	// tickerPrice.Last = tick.Last
	// tickerPrice.Volume = tick.Volume24H
	// tickerPrice.High = tick.High
	// ticker.ProcessTicker(b.GetName(), p, tickerPrice, assetType)
	// return ticker.GetTicker(b.Name, p, assetType)
	return ticker.Price{}, errors.New("REST NOT SUPPORTED")
}

// FetchTicker returns the ticker for a currency pair
func (b *BTCC) FetchTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	// tickerNew, err := ticker.GetTicker(b.GetName(), p, assetType)
	// if err != nil {
	// 	return b.UpdateTicker(p, assetType)
	// }
	// return tickerNew, nil
	return ticker.Price{}, errors.New("REST NOT SUPPORTED")
}

// FetchOrderbook returns the orderbook for a currency pair
func (b *BTCC) FetchOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	// ob, err := orderbook.GetOrderbook(b.GetName(), p, assetType)
	// if err != nil {
	// 	return b.UpdateOrderbook(p, assetType)
	// }
	// return ob, nil
	return orderbook.Base{}, errors.New("REST NOT SUPPORTED")
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *BTCC) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	// var orderBook orderbook.Base
	// orderbookNew, err := b.GetOrderBook(exchange.FormatExchangeCurrency(b.GetName(), p).String(), 100)
	// if err != nil {
	// 	return orderBook, err
	// }

	// for x := range orderbookNew.Bids {
	// 	data := orderbookNew.Bids[x]
	// 	orderBook.Bids = append(orderBook.Bids, orderbook.Item{Price: data[0], Amount: data[1]})
	// }

	// for x := range orderbookNew.Asks {
	// 	data := orderbookNew.Asks[x]
	// 	orderBook.Asks = append(orderBook.Asks, orderbook.Item{Price: data[0], Amount: data[1]})
	// }

	// orderbook.ProcessOrderbook(b.GetName(), p, orderBook, assetType)
	// return orderbook.GetOrderbook(b.Name, p, assetType)
	return orderbook.Base{}, errors.New("REST NOT SUPPORTED")
}

// GetExchangeAccountInfo : Retrieves balances for all enabled currencies for
// the Kraken exchange - TODO
func (b *BTCC) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	// var response exchange.AccountInfo
	// response.ExchangeName = b.GetName()
	// return response, nil
	return exchange.AccountInfo{}, errors.New("REST NOT SUPPORTED")
}

// GetExchangeFundTransferHistory returns funding history, deposits and
// withdrawals
func (b *BTCC) GetExchangeFundTransferHistory() ([]exchange.FundHistory, error) {
	// var fundHistory []exchange.FundHistory
	// return fundHistory, errors.New("not supported on exchange")
	return nil, errors.New("REST NOT SUPPORTED")
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *BTCC) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	// var resp []exchange.TradeHistory

	// return resp, errors.New("trade history not yet implemented")
	return nil, errors.New("REST NOT SUPPORTED")
}

// SubmitExchangeOrder submits a new order
func (b *BTCC) SubmitExchangeOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// ModifyExchangeOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *BTCC) ModifyExchangeOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// CancelExchangeOrder cancels an order by its corresponding ID number
func (b *BTCC) CancelExchangeOrder(orderID int64) error {
	return errors.New("not yet implemented")
}

// CancelAllExchangeOrders cancels all orders associated with a currency pair
func (b *BTCC) CancelAllExchangeOrders() error {
	return errors.New("not yet implemented")
}

// GetExchangeOrderInfo returns information on a current open order
func (b *BTCC) GetExchangeOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, errors.New("not yet implemented")
}

// GetExchangeDepositAddress returns a deposit address for a specified currency
func (b *BTCC) GetExchangeDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawCryptoExchangeFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *BTCC) WithdrawCryptoExchangeFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCC) WithdrawFiatExchangeFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCC) WithdrawFiatExchangeFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *BTCC) GetWebsocket() (*exchange.Websocket, error) {
	return b.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *BTCC) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return b.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (b *BTCC) GetWithdrawCapabilities() uint32 {
	return b.GetWithdrawPermissions()
}
