C:\Guru\pkg\mod\github.com\zerodha\gokiteconnect\v4@v4.0.9\gtt.go
Line 126 - Change to ProductNRML

orders = append(orders, Order{
			Exchange:        o.Exchange,
			TradingSymbol:   o.Tradingsymbol,
			TransactionType: o.TransactionType,
			Quantity:        o.Trigger.Quantities()[i],
			Price:           o.Trigger.LimitPrices()[i],
			OrderType:       OrderTypeLimit,
			Product:         ProductNRML,
		})