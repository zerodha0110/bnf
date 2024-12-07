package order

import (
	"fmt"
	myTicker "kite/ticker"
	"kite/types"
	"kite/util"
	"log"
	"math"
	"time"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

var orderCount int
var orderIDToPrice map[string]float64

func init() {
	orderIDToPrice = make(map[string]float64, 1)
}

func GetUserMargin(kc *kiteconnect.Client) float64 {
	margin := 0.0
	userMargins, error123 := kc.GetUserMargins()
	if error123 == nil {
		margin = userMargins.Equity.Available.LiveBalance
	}
	return margin
}

func WaitForOrderToExecute(kc *kiteconnect.Client, orderId string, test bool) (bool, float64, error, string) {
	if test {
		return true, 0.0, nil, kiteconnect.OrderStatusComplete
	}

	retryCount := 10
	var err error = nil
	status := ""
	averagePrice := 0.0
	executed := false
	fmt.Println("Wait for order execution...", orderId)
	/*
		if util.GetEnvConfig().TestRun {
			return true, averagePrice, nil, kiteconnect.OrderStatusComplete
		}
	*/

	for retryCount != 0 {
		status, averagePrice, err, _ = GetOrderStatus(kc, orderId)
		if err != nil {
			fmt.Printf("error fetching order status: %v", err)
			retryCount = retryCount - 1
		} else {
			if status == kiteconnect.OrderStatusComplete {
				executed = true
				break
			} else if status == kiteconnect.OrderStatusCancelled {
				fmt.Println("The order status is cancelled:WaitForOrderToExecute1")
				break
			} else if status == kiteconnect.OrderStatusRejected {
				fmt.Println("The order status is rejected:WaitForOrderToExecute1")
				break
			}
			retryCount = 10 //Again start from beginning
		}
		time.Sleep(time.Second * 1)
	}
	//TODO -- if order is not executed for more than 5 mins, then get currentTick and execute at currentTick....important...
	fmt.Println("OrderExecuted:Return", executed, averagePrice, orderId, err)
	return executed, averagePrice, err, status

}

func CancelOrder(kc *kiteconnect.Client, orderId string, waitTime int) (bool, error) {

	time.Sleep(time.Second * time.Duration(waitTime))
	retryCount := 10
	var err error = nil
	status := ""
	var orderObj kiteconnect.Order
	fmt.Println("Cancel order if it is not yet executed.", orderId)
	/*
		if util.GetEnvConfig().TestRun {
			return false, nil
		}
	*/
	for retryCount != 0 {
		status, _, err, orderObj = GetQuickOrderStatus(kc, orderId)
		fmt.Println("The order status  is ", status)
		fmt.Println("The order error is ", err)
		if err != nil {
			fmt.Printf("error fetching order status to cancel: %v", err)
			retryCount = retryCount - 1
			time.Sleep(time.Second * 1)
		} else {
			//TODO check for pending or inprogress instead of != complete
			if status != kiteconnect.OrderStatusComplete && status != kiteconnect.OrderStatusCancelled {
				fmt.Println("Order is not complete nor cancelled. Try to cancel it now !!!")
				orderResp, err := kc.CancelOrder("regular", orderId, &orderObj.ParentOrderID)
				if err == nil {
					// Wait for a sec before checking status.
					time.Sleep(2 * time.Second)
					status, _, _, _ = GetOrderStatus(kc, orderResp.OrderID)
					fmt.Printf("Current status %s", status)
					if status == kiteconnect.OrderStatusCancelled {
						return true, nil
					} else {
						return false, nil
					}
				} else {
					fmt.Println("Error while cancelling ", err)
					return false, nil
				}
			} else {
				fmt.Printf("Not cancelling.. just going out... %s", status)
				fmt.Println()
				return false, nil
			}
		}
	}
	return false, nil
}

func GetOrderStatus(kc *kiteconnect.Client, orderId string) (string, float64, error, kiteconnect.Order) {
	retryCount := types.GTTRetryCount
	var err error = nil
	var orderResp []kiteconnect.Order
	var status string = ""
	var averagePrice float64 = 0.0
	var order1 kiteconnect.Order

	/*
		if util.GetEnvConfig().TestRun || util.GetEnvConfig().Simulation {
			return kiteconnect.OrderStatusComplete, orderIDToPrice[orderId], nil, order1
		}
	*/

	for retryCount != 0 {
		time.Sleep(time.Second * 2)
		orderResp, err = kc.GetOrderHistory(orderId)
		if err != nil {
			fmt.Printf("error fetching order History: %v", err)
			retryCount = retryCount - 1
		} else {
			for _, order := range orderResp {
				if order.OrderID == "" {
					log.Printf("Error while fetching order id in order history. %v", err)
				} else if order.OrderID == orderId {
					status = order.Status
					if status == kiteconnect.OrderStatusComplete {
						averagePrice = order.AveragePrice
						return status, averagePrice, nil, order
					} else if status == kiteconnect.OrderStatusCancelled {
						return status, 0.0, nil, order
					} else if status == kiteconnect.OrderStatusRejected {
						return status, 0.0, nil, order
					}
				}
			}
		}
	}
	return status, averagePrice, err, order1
}

func GetQuickOrderStatus(kc *kiteconnect.Client, orderId string) (string, float64, error, kiteconnect.Order) {
	retryCount := types.GTTRetryCount
	var err error = nil
	var orderResp []kiteconnect.Order
	var status string = ""
	var averagePrice float64 = 0.0
	var order1 kiteconnect.Order

	/*
		if util.GetEnvConfig().TestRun || util.GetEnvConfig().Simulation {
			return kiteconnect.OrderStatusComplete, orderIDToPrice[orderId], nil, order1
		}
	*/
	var order kiteconnect.Order
	for retryCount != 0 {
		time.Sleep(time.Second * 2)
		retryCount = retryCount - 1
		orderResp, err = kc.GetOrderHistory(orderId)
		if err != nil {
			fmt.Printf("error fetching order History: %v", err)
		} else {
			for _, order = range orderResp {
				if order.OrderID == "" {
					log.Printf("Error while fetching order id in order history. %v", err)
				} else if order.OrderID == orderId {
					status = order.Status
					if status == kiteconnect.OrderStatusComplete {
						averagePrice = order.AveragePrice
					} else {
						averagePrice = 0.0
					}
				}
			}
			return status, averagePrice, nil, order
		}
	}
	return status, averagePrice, err, order1
}

func PlaceOrderWithBuffer(kc *kiteconnect.Client, transactionType string, lotSize, limitPrice float64) (string, error) {
	if transactionType == kiteconnect.TransactionTypeBuy {
		limitPrice = limitPrice + types.TransactionBuffer // Buy for little more
	} else if transactionType == kiteconnect.TransactionTypeSell {
		limitPrice = limitPrice - types.TransactionBuffer // Sell for little less
	}
	return PlaceOrder(kc, transactionType, lotSize, limitPrice)
}

func PlaceOrder(kc *kiteconnect.Client, transactionType string, lotSize, limitPrice float64) (string, error) {

	retryCount := types.GTTRetryCount
	var err error = nil

	limitPrice = float64(int(limitPrice))
	/*
		if util.GetEnvConfig().TestRun || util.GetEnvConfig().Simulation {
			orderCount = orderCount + 1
			tempOrderId := "abcde-" + strconv.Itoa(orderCount)
			orderIDToPrice[tempOrderId] = limitPrice
			return tempOrderId, nil
		}
	*/
	for retryCount != 0 {
		params := kiteconnect.OrderParams{
			Exchange:          "NFO",
			Tradingsymbol:     types.BankNiftySymbol,
			Product:           kiteconnect.ProductNRML,
			OrderType:         kiteconnect.OrderTypeLimit,
			TransactionType:   transactionType,
			Quantity:          int(lotSize),
			DisclosedQuantity: int(lotSize),
			Price:             limitPrice,
		}
		orderResponse, err := kc.PlaceOrder("regular", params)
		if err != nil {
			fmt.Println()
			fmt.Printf("Error while placing order. %v", err)
			retryCount = retryCount - 1
			time.Sleep(1 * time.Second)
		} else if orderResponse.OrderID == "" {
			fmt.Printf("No order id returned. Error %v", err)
			retryCount = retryCount - 1
			time.Sleep(1 * time.Second)
		} else {
			fmt.Println("Order placed", transactionType, lotSize, limitPrice, orderResponse.OrderID)
			return orderResponse.OrderID, nil
		}
	}

	return "", err
}

func PlaceOrderSharesWithTag(kc *kiteconnect.Client, transactionType string, lotSize, limitPrice float64, tickerId string, test bool, tag string) (string, error) {

	if test {
		return "123", nil
	}
	retryCount := types.GTTRetryCount
	var err error = nil

	limitPrice = float64(int(limitPrice))
	limitPrice = math.Round(limitPrice*10) / 10
	/*
		if util.GetEnvConfig().TestRun || util.GetEnvConfig().Simulation {
			orderCount = orderCount + 1
			tempOrderId := "abcde-" + strconv.Itoa(orderCount)
			orderIDToPrice[tempOrderId] = limitPrice
			return tempOrderId, nil
		}
	*/

	for retryCount != 0 {
		params := kiteconnect.OrderParams{
			Exchange:          "NFO",
			Tradingsymbol:     tickerId,
			Product:           kiteconnect.ProductNRML,
			OrderType:         kiteconnect.OrderTypeLimit,
			TransactionType:   transactionType,
			Quantity:          int(lotSize),
			DisclosedQuantity: int(lotSize),
			Price:             limitPrice,
			Tag:               tag,
		}
		orderResponse, err := kc.PlaceOrder("regular", params)
		if err != nil {
			fmt.Println()
			fmt.Printf("Error while placing order. %v", err)
			retryCount = retryCount - 1
			time.Sleep(1 * time.Second)
		} else if orderResponse.OrderID == "" {
			fmt.Printf("No order id returned. Error %v", err)
			retryCount = retryCount - 1
			time.Sleep(1 * time.Second)
		} else {
			fmt.Println("Order placed", transactionType, lotSize, limitPrice, orderResponse.OrderID)
			return orderResponse.OrderID, nil
		}

	}

	return "", err
}

func PlaceOrderSharesWithTagInNSE(kc *kiteconnect.Client, transactionType string, lotSize, limitPrice float64, tickerId string, test bool, tag string, exchange string) (string, error) {

	fmt.Println("tickerId to PlaceOrderSharesWithTagInNSE: ", tickerId, transactionType, exchange)
	if test {
		return "123", nil
	}
	retryCount := types.GTTRetryCount
	var err error = nil

	//limitPrice = float64(int(limitPrice))
	limitPrice = math.Round(limitPrice*10) / 10
	/*
		if util.GetEnvConfig().TestRun || util.GetEnvConfig().Simulation {
			orderCount = orderCount + 1
			tempOrderId := "abcde-" + strconv.Itoa(orderCount)
			orderIDToPrice[tempOrderId] = limitPrice
			return tempOrderId, nil
		}
	*/

	product := kiteconnect.ProductCNC
	if exchange == "NFO" {
		product = kiteconnect.ProductNRML
	}
	for retryCount != 0 {
		params := kiteconnect.OrderParams{
			Exchange:          exchange,
			Tradingsymbol:     tickerId,
			Product:           product,
			OrderType:         kiteconnect.OrderTypeLimit,
			TransactionType:   transactionType,
			Quantity:          int(lotSize),
			DisclosedQuantity: int(lotSize),
			Price:             limitPrice,
			Tag:               tag,
		}
		orderResponse, err := kc.PlaceOrder("regular", params)
		if err != nil {
			fmt.Println()
			fmt.Printf("Error while placing order. %v", err)
			retryCount = retryCount - 1
			time.Sleep(1 * time.Second)
		} else if orderResponse.OrderID == "" {
			fmt.Printf("No order id returned. Error %v", err)
			retryCount = retryCount - 1
			time.Sleep(1 * time.Second)
		} else {
			fmt.Println("Order placed", transactionType, lotSize, limitPrice, orderResponse.OrderID)
			return orderResponse.OrderID, nil
		}

	}

	return "", err
}

func PlaceOrderShares(kc *kiteconnect.Client, transactionType string, lotSize, limitPrice float64, tickerId string, test bool) (string, error) {

	if test {
		return "123", nil
	}
	retryCount := types.GTTRetryCount
	var err error = nil

	limitPrice = float64(int(limitPrice))
	limitPrice = math.Round(limitPrice*10) / 10
	/*
		if util.GetEnvConfig().TestRun || util.GetEnvConfig().Simulation {
			orderCount = orderCount + 1
			tempOrderId := "abcde-" + strconv.Itoa(orderCount)
			orderIDToPrice[tempOrderId] = limitPrice
			return tempOrderId, nil
		}
	*/

	for retryCount != 0 {
		params := kiteconnect.OrderParams{
			Exchange:          "NFO",
			Tradingsymbol:     tickerId,
			Product:           kiteconnect.ProductNRML,
			OrderType:         kiteconnect.OrderTypeLimit,
			TransactionType:   transactionType,
			Quantity:          int(lotSize),
			DisclosedQuantity: int(lotSize),
			Price:             limitPrice,
		}
		orderResponse, err := kc.PlaceOrder("regular", params)
		if err != nil {
			fmt.Println()
			fmt.Printf("Error while placing order. %v", err)
			retryCount = retryCount - 1
			time.Sleep(1 * time.Second)
		} else if orderResponse.OrderID == "" {
			fmt.Printf("No order id returned. Error %v", err)
			retryCount = retryCount - 1
			time.Sleep(1 * time.Second)
		} else {
			fmt.Println("Order placed", transactionType, lotSize, limitPrice, orderResponse.OrderID)
			return orderResponse.OrderID, nil
		}

	}

	return "", err
}

func PlaceOrderAndMonitor(kc *kiteconnect.Client, transactionType string, qty float64, limitPrice float64, wait bool, test bool) (string, float64) {
	if limitPrice == 0.0 {
		limitPrice = myTicker.GetCurrentTick()
	}
	orderId, err := PlaceOrderWithBuffer(kc, transactionType, qty, limitPrice)
	if err != nil || orderId == "" {
		log.Fatalf("Cannot place %s order %v", transactionType, err)
	}
	if wait {
		util.MyPrintf("Start monitoring order status:%s", orderId)
		executed, avgExitPrice, err, _ := WaitForOrderToExecute(kc, orderId, test)
		if err != nil || !executed {
			fmt.Println("Executed !!!! ??????", executed)
			log.Fatalf("Error while checking for order status %v", err)
		}
		return orderId, avgExitPrice
	}
	return orderId, 0.0
}

func GetOrderMargin(kc *kiteconnect.Client, transactionType string, lotSize float64, tickerId string, test bool) (float64, error) {
	if test {
		return 0.0, nil
	}

	retryCount := 1
	var err error = nil
	margin := 0.0

	for retryCount != 0 {
		params := kiteconnect.OrderMarginParam{
			Exchange:        "NFO",
			Tradingsymbol:   tickerId,
			Product:         kiteconnect.ProductNRML,
			OrderType:       kiteconnect.OrderTypeLimit,
			TransactionType: transactionType,
			Quantity:        lotSize,
			Variety:         kiteconnect.VarietyRegular,
		}

		paramsArray := make([]kiteconnect.OrderMarginParam, 1)
		paramsArray[0] = params

		marginParams := kiteconnect.GetMarginParams{
			Compact:     true,
			OrderParams: paramsArray,
		}
		marginResponse, err := kc.GetOrderMargins(marginParams)
		fmt.Println("AFASDFASDFASFASDF", marginResponse)

		if err != nil {
			fmt.Println()
			fmt.Printf("Error while getting margin response. %v", err)
			retryCount = retryCount - 1
			time.Sleep(1 * time.Second)
		} else {
			margin = marginResponse[0].Total
			fmt.Printf("The margin is %f", margin)
			return margin, nil
		}
	}
	return margin, err

}
