package internal

import (
	"fmt"
	"log"
)

type PgFgaHandler struct {
	config FgaConfig
	pg     *bitvavo.Bitvavo
	ldap   BvvMarkets
}

func NewPgFgaHandler() (bh *PgFgaHandler, err error) {

	config, err := NewConfig()

	if err != nil {
		return bh, err
	}
	connection := bitvavo.Bitvavo{
		ApiKey:       config.Api.Key,
		ApiSecret:    config.Api.Secret,
		RestUrl:      "https://api.bitvavo.com/v2",
		WsUrl:        "wss://ws.bitvavo.com/v2/",
		AccessWindow: 10000,
		Debugging:    config.Api.Debug,
	}
	return &PgFgaHandler{
		config:     config,
		connection: &connection,
	}, nil
}

func (bh PgFgaHandler) Handle() {
	markets, err := bh.GetMarkets(false)
	if err != nil {
		log.Fatalf("Error occurred on getting markets: %e", err)
	}
	for _, market := range markets {
		if market.To != bh.config.Fiat {
			// This probably is a reverse market. Skipping.
			continue
		}
		if market.mah != nil {
			expectedRate, err := market.GetExpectedRate()
			if err != nil {
				log.Fatalf("Error occurred on getting GetExpectedRate for market %s: %e", market.Name(), err)
			}
			var direction string
			var percent decimal.Decimal
			hundred := decimal.NewFromInt(100)
			if expectedRate.GreaterThan(market.Price) {
				direction = "under"
				percent = hundred.Sub(market.Price.Div(expectedRate).Mul(hundred))
			} else {
				direction = "over"
				percent = hundred.Sub(expectedRate.Div(market.Price).Mul(hundred))
			}
			fmt.Printf("%s is %s%% %srated (expected %s vs actual %s)\n", market.Name(), percent.Round(2),
				direction, expectedRate.Round(2), market.Price)
			bw, err := market.GetBandWidth()
			if err != nil {
				log.Fatalf("Error occurred on getting GetBandWidth for market %s: %e", market.Name(), err)
			}
			fmt.Printf("%s bandwidth is between -%s%% and +%s%%.\n", market.Name(), bw.GetMinPercent().Round(2),
				bw.GetMaxPercent().Round(2))
		}
		if market.Min.GreaterThan(decimal.Zero) && market.Max.LessThan(market.Total()) {
			err := bh.Sell(*market, market.Total().Sub(market.Min))
			if err != nil {
				log.Fatalf("Error occurred while selling %s: %e", market.Name(), err)
			}
		}
	}
}

func (bh PgFgaHandler) GetBvvTime() (time bitvavo.Time, err error) {
	return bh.connection.Time()
}

func (bh PgFgaHandler) GetRemainingLimit() (limit int) {
	return bh.connection.GetRemainingLimit()
}

func (bh *PgFgaHandler) getPrices(reset bool) (prices map[string]decimal.Decimal, err error) {
	if len(bh.prices) > 0 && !reset {
		return bh.prices, nil
	}
	bh.prices = make(map[string]decimal.Decimal)
	prices = make(map[string]decimal.Decimal)
	tickerPriceResponse, tickerPriceErr := bh.connection.TickerPrice(bvvOptions{})
	if tickerPriceErr != nil {
		fmt.Println(tickerPriceErr)
	} else {
		for _, price := range tickerPriceResponse {
			prices[price.Market], err = decimal.NewFromString(price.Price)
			if err != nil {
				return bh.prices, err
			}
		}
	}
	bh.prices = prices
	return prices, err
}

func (bh *PgFgaHandler) GetMarkets(reset bool) (markets BvvMarkets, err error) {
	if len(bh.markets) > 0 && !reset {
		return bh.markets, nil
	}
	bh.markets = make(BvvMarkets)
	markets = make(BvvMarkets)

	_, err = bh.getPrices(false)
	if err != nil {
		return markets, err
	}
	balanceResponse, balanceErr := bh.connection.Balance(bvvOptions{})
	if balanceErr != nil {
		return markets, err
	} else {
		for _, b := range balanceResponse {
			if b.Symbol == bh.config.Fiat {
				continue
			}
			_, err := NewBvvMarket(bh, b.Symbol, bh.config.Fiat, b.Available, b.InOrder)
			if mErr, ok  := err.(MarketNotInConfigError); ok {
				if bh.config.Debug {
					fmt.Printf("%s.\n", mErr.Error())
				}
				continue
			}
			if err != nil {
				return bh.markets, err
			}
		}
	}
	return bh.markets, nil
}

func (bh PgFgaHandler) Sell(market BvvMarket, amount decimal.Decimal) (err error) {
	if !bh.config.ActiveMode {
		fmt.Printf("We should sell %s: %s\n", market.Name(), amount)
		bh.PrettyPrint(market.inverse)
		return nil
	}
	fmt.Printf("I am selling %s: %s\n", market.Name(), amount)
	bh.PrettyPrint(market.inverse)
	placeOrderResponse, err := bh.connection.PlaceOrder(
		market.Name(),
		"sell",
		"market",
		bvvOptions{"amount": amount.String()})
	if err != nil {
		return err
	} else {
		bh.PrettyPrint(placeOrderResponse)
	}
	return nil
}

//func (bh PgFgaHandler) GetMarkets() (err error) {
//	marketsResponse, marketsErr := bh.connection.Markets(bvvOptions{})
//	if marketsErr != nil {
//		fmt.Println(marketsErr)
//	} else {
//		for _, value := range marketsResponse {
//			err = bh.PrettyPrint(value)
//			if err != nil {
//				log.Printf("Error on PrettyPrint: %e", err)
//			}
//		}
//	}
//	return nil
//}

func (bh PgFgaHandler) GetAssets() (err error) {
	assetsResponse, assetsErr := bh.connection.Assets(bvvOptions{})
	if assetsErr != nil {
		fmt.Println(assetsErr)
	} else {
		for _, value := range assetsResponse {
			bh.PrettyPrint(value)
		}
	}
	return nil
}

func (bh PgFgaHandler) PrettyPrint(v interface{}) {
	if bh.config.Debug {
		err := PrettyPrint(v)
		if err != nil {
			log.Printf("Error on PrettyPrint: %e", err)
		}
	}
}

//fmt.Println("Book")
//bookResponse, bookErr := bitvavo.Book("BTC-EUR", bvvOptions{})
//if bookErr != nil {
// fmt.Println(bookErr)
//} else {
// PrettyPrint(bookResponse)
//}

// publicTradesResponse, publicTradesErr := bitvavo.PublicTrades("BTC-EUR", bvvOptions{})
// if publicTradesErr != nil {
//   fmt.Println(publicTradesErr)
// } else {
//   for _, trade := range publicTradesResponse {
//     PrettyPrint(trade)
//   }
// }

// candlesResponse, candlesErr := bitvavo.Candles("BTC-EUR", "1h", bvvOptions{})
// if candlesErr != nil {
//   fmt.Println(candlesErr)
// } else {
//   for _, candle := range candlesResponse {
//     PrettyPrint(candle)
//   }
// }

// tickerPriceResponse, tickerPriceErr := bitvavo.TickerPrice(bvvOptions{})
// if tickerPriceErr != nil {
//   fmt.Println(tickerPriceErr)
// } else {
//   for _, price := range tickerPriceResponse {
//     PrettyPrint(price)
//   }
// }

// tickerBookResponse, tickerBookErr := bitvavo.TickerBook(bvvOptions{})
// if tickerBookErr != nil {
//   fmt.Println(tickerBookErr)
// } else {
//   for _, book := range tickerBookResponse {
//     PrettyPrint(book)
//   }
// }

// ticker24hResponse, ticker24hErr := bitvavo.Ticker24h(bvvOptions{})
// if ticker24hErr != nil {
//   fmt.Println(ticker24hErr)
// } else {
//   for _, ticker := range ticker24hResponse {
//     PrettyPrint(ticker)
//   }
// }

// placeOrderResponse, placeOrderErr := bitvavo.PlaceOrder(
//   "BTC-EUR",
//   "buy",
//   "limit",
//   bvvOptions{"amount": "0.3", "price": "2000"})
// if placeOrderErr != nil {
//   fmt.Println(placeOrderErr)
// } else {
//   PrettyPrint(placeOrderResponse)
// }

// placeOrderResponse, placeOrderErr := bitvavo.PlaceOrder(
//   "BTC-EUR",
//   "sell",
//   "stopLoss",
//   bvvOptions{"amount": "0.1", "triggerType": "price", "triggerReference": "lastTrade", "triggerAmount": "5000"})
// if placeOrderErr != nil {
//   fmt.Println(placeOrderErr)
// } else {
//   PrettyPrint(placeOrderResponse)
// }

// updateOrderResponse, updateOrderErr := bitvavo.UpdateOrder("BTC-EUR", "68c72b7a-2cf5-4516-8915-703a5d38c77e", bvvOptions{"amount": "0.4"})
// if updateOrderErr != nil {
//   fmt.Println(updateOrderErr)
// } else {
//   PrettyPrint(updateOrderResponse)
// }

// getOrderResponse, getOrderErr := bitvavo.GetOrder("BTC-EUR", "68c72b7a-2cf5-4516-8915-703a5d38c77e")
// if getOrderErr != nil {
//   fmt.Println(getOrderErr)
// } else {
//   PrettyPrint(getOrderResponse)
// }

// cancelOrderResponse, cancelOrderErr := bitvavo.CancelOrder("BTC-EUR", "68c72b7a-2cf5-4516-8915-703a5d38c77e")
// if cancelOrderErr != nil {
//   fmt.Println(cancelOrderErr)
// } else {
//   PrettyPrint(cancelOrderResponse)
// }

//fmt.Println("Orders")
//getOrdersResponse, getOrdersErr := bitvavo.GetOrders("BTC-EUR", bvvOptions{})
//if getOrdersErr != nil {
//  fmt.Println(getOrdersErr)
//} else {
//  for _, order := range getOrdersResponse {
//    PrettyPrint(order)
//  }
//}

// cancelOrdersResponse, cancelOrdersErr := bitvavo.CancelOrders(bvvOptions{"market": "BTC-EUR"})
// if cancelOrdersErr != nil {
//   fmt.Println(cancelOrdersErr)
// } else {
//   for _, order := range cancelOrdersResponse {
//     PrettyPrint(order)
//   }
// }

// ordersOpenResponse, ordersOpenErr := bitvavo.OrdersOpen(bvvOptions{"market": "BTC-EUR"})
// if ordersOpenErr != nil {
//   fmt.Println(ordersOpenErr)
// } else {
//   for _, order := range ordersOpenResponse {
//     PrettyPrint(order)
//   }
// }

// tradesResponse, tradesErr := bitvavo.Trades("BTC-EUR", bvvOptions{})
// if tradesErr != nil {
//   fmt.Println(tradesErr)
// } else {
//   for _, trade := range tradesResponse {
//     PrettyPrint(trade)
//   }
// }

// accountResponse, accountErr := bitvavo.Account()
// if accountErr != nil {
//   fmt.Println(accountErr)
// } else {
//   PrettyPrint(accountResponse)
// }

// depositAssetsResponse, depositAssetsErr := bitvavo.DepositAssets("BTC")
// if depositAssetsErr != nil {
//   fmt.Println(depositAssetsErr)
// } else {
//   PrettyPrint(depositAssetsResponse)
// }

// withdrawAssetsResponse, withdrawAssetsErr := bitvavo.WithdrawAssets("BTC", "1", "BitcoinAddress", bvvOptions{})
// if withdrawAssetsErr != nil {
//   fmt.Println(withdrawAssetsErr)
// } else {
//   PrettyPrint(withdrawAssetsResponse)
// }

// depositHistoryResponse, depositHistoryErr := bitvavo.DepositHistory(bvvOptions{})
// if depositHistoryErr != nil {
//   fmt.Println(depositHistoryErr)
// } else {
//   for _, deposit := range depositHistoryResponse {
//     PrettyPrint(deposit)
//   }
// }

// withdrawalHistoryResponse, withdrawalHistoryErr := bitvavo.WithdrawalHistory(bvvOptions{})
// if withdrawalHistoryErr != nil {
//   fmt.Println(withdrawalHistoryErr)
// } else {
//   for _, withdrawal := range withdrawalHistoryResponse {
//     PrettyPrint(withdrawal)
//   }
// }
