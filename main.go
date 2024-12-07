package main

import (
	"fmt"
	"kite/order"
	"kite/strategy/equityVolume"

	myTicker "kite/ticker"
	"kite/types"
	"kite/util"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
	kiteticker "github.com/zerodha/gokiteconnect/v4/ticker"
)

const (
	//Guru
	apiKey    string = "pshndp9jhkxo36ic"
	apiSecret string = "7jttbhol2tohvv4rrej0b03v3sb56jf1"
)

var (
	kc *kiteconnect.Client = nil

	alreadyEntryGTTCreated bool = true
	alreadyOrderPlaced     bool = true
	alreadyOrderCreated    bool = true

	buyEntryGTTId      int     = 0
	sellEntryGTTId     int     = 0
	triggeredPosition  string  = ""  // which one was triggered ? buy or sell ?
	positionEntryPrice float64 = 0.0 // Price at which order was executed
	transactionType    string  = ""  // Type of order executed - buy or sell
	exit                       = false
)

var (
	shutdownChannel        chan os.Signal
	positionEnteredChannel chan struct{}
)

func main() {

	fmt.Println("OM")

	envConf := util.GetEnvConfig()

	shutdownChannel = make(chan os.Signal, 1) //TODO use this to shutdown at 3:30
	positionEnteredChannel = make(chan struct{}, 1)

	if envConf.GTTConf.PlaceOrder {
		alreadyOrderCreated = false
	} else {
		alreadyOrderCreated = true
	}

	// Create a new Kite connect instance
	kc = kiteconnect.New(apiKey)

	// Login URL from which request token can be obtained
	fmt.Println(kc.GetLoginURL())

	if !envConf.TestRun {

		// Get user details and access token
		data, err := kc.GenerateSession(envConf.RequestToken, apiSecret)
		if err != nil {
			fmt.Printf("Error while generating session. Will exit now!!!: %v", err)
			return
		}

		// Set access token
		kc.SetAccessToken(data.AccessToken)

		// Create new Kite ticker instance
		fmt.Println("Starting ticker for", envConf.CFWSConf.TickerName)
		// Also set dayHigh and dayLow
		myTicker.SetDayHighLow(envConf.CFWConf.DayHigh, envConf.CFWConf.DayLow)
		if envConf.Strategy == types.STRATEGY_CANDLE_FLOW_WAVE_SHARE || envConf.Strategy == types.STRATEGY_CANDLE_HEIGHT_SHARE {
			if strings.HasPrefix(envConf.CFWSConf.TickerName, "EQUITY-VOLUME") {

				bnfDetailsList := equityVolume.GetBNFDetails()

				tickerIds := make([]uint32, 0)
				for _, obj := range bnfDetailsList {
					temp, err := strconv.Atoi(obj.Id)
					if err == nil {
						id := uint32(temp)
						tickerIds = append(tickerIds, id)
					}
				}
				myTicker.SetTickerIDArray(tickerIds)
			} else {
				myTicker.SetTickerID(uint32(envConf.CFWSConf.TickerId))
			}
		}
		ticker := kiteticker.New(apiKey, data.AccessToken)

		go myTicker.StartTicker(ticker) // Keep getting latest BNF Ticks
	}

	/*
		fmt.Println("current strategy is ", envConf.Strategy, strconv.FormatUint(uint64(envConf.CFWSConf.TickerId), 10), envConf.CFWSConf.TickerName)

		requiredMargin1, err1213 := order.GetOrderMargin(kc, kiteconnect.TransactionTypeBuy, envConf.CFWSConf.LotSize, envConf.CFWSConf.TickerName)

		fmt.Println(requiredMargin1, err1213)
	*/

	//Wait till 9:00
	fmt.Println("111111111111111111111111111111111111111111111111111111")

	testRun, _ := strconv.ParseBool(os.Getenv("TEST_RUN"))

	if !testRun {
		dt := time.Now()
		for dt.Hour() < 9 || dt.Hour() > 16 {
			time.Sleep(20 * time.Minute)
			dt = time.Now()
		}
	}

	/*
		// Wait till 9:20
		dt = time.Now()
		for dt.Hour() < 10 && dt.Minute() < 20 {
			//	fmt.Println("0000000000000000000000000000000000000000000", dt.Hour(), dt.Minute())
			time.Sleep(1 * time.Minute)
			dt = time.Now()
		}
	*/

	fmt.Println("STTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTAAAAAARRRRRRRRRRRRRRTTTTTTTTTTTTTTTTTTT")
	time.Sleep(6 * time.Second) // Sleep for 6 seconds so that we have a valid Tick before we start our strategy
	if envConf.Strategy == types.STRATEGY_GTT || envConf.Strategy == types.STRATEGY_TWO_LEG_EXIT {
		go placeOrderAndMonitorForGTT() // Monitor ticker, place order and monitor till execution
	} else if envConf.Strategy == types.STRATEGY_CANDLE_HEIGHT_SHARE {
		if strings.HasPrefix(envConf.CFWSConf.TickerName, "EQUITY-VOLUME") {
			fmt.Println("Its equity volume.. The finals !!!!")
			equityVolume.ExecuteCandleHeightShareStrategy(kc)
		}
	} else {
		fmt.Println("Unknown Strategy")
	}

	// Notify shutdownChannel when an interrupt or SIGTERM signal is received
	signal.Notify(shutdownChannel, syscall.SIGINT, syscall.SIGTERM)
	go exitAt3(shutdownChannel)

	<-shutdownChannel

	myMarketMaker := equityVolume.GetMarketMakerSetup()
	//myMarketMaker.CreateReport()
	fmt.Println("Now create GTT for open trades")
	myMarketMaker.PlaceGTTForPendingPositions()
	fmt.Println("NOw printing finaly summay of the day at the exit")
	myMarketMaker.PrintDaysFinalSummary()
}

func placeOrderAndMonitorForGTT() {

	envConf := util.GetEnvConfig()
	// TODO - combine position_exists and place_order to one variable. Both cannot be true at same time.
	if envConf.GTTConf.PlaceOrder {
		fmt.Println("Placing order")
	} else {
		fmt.Println("No need to place order ... Exit !!!!")
		<-positionEnteredChannel // Wait till Position Created is done..
		return
	}

	done := false

	for !done {
		// Get BankNifty tick
		bnfDetails := myTicker.GetBNFDetails()
		currentTick := bnfDetails.CurrentTick.LastPrice
		placeOrder := false

		// First place buy Order
		if currentTick >= envConf.GTTConf.PlaceBuyOrderEntryPrice {
			placeOrder = true
			transactionType = kiteconnect.TransactionTypeBuy // Type of order executed - buy or sell
		} else if currentTick <= envConf.GTTConf.PlaceSellOrderEntryPrice {
			placeOrder = true
			transactionType = kiteconnect.TransactionTypeSell // Type of order executed - buy or sell
		} else {
			fmt.Println("Waiting, since currentTick doesn't match entry price", currentTick)
			time.Sleep(5 * time.Second)
		}
		if placeOrder {
			util.MyPrintf("Start place order api %s, %f, %f, %f", transactionType, envConf.GTTConf.LotSize, currentTick, positionEntryPrice)
			orderId, err := order.PlaceOrderWithBuffer(kc, transactionType, envConf.GTTConf.LotSize, positionEntryPrice)
			if err != nil || orderId == "" {
				log.Fatalf("Cannot place order %v", err)
			}
			fmt.Println("Start monitoring order status")
			executed, avgPrice, err, _ := order.WaitForOrderToExecute(kc, orderId, true)
			if err != nil {
				log.Fatalf("Error while checking for order status %v", err)
			}
			if executed {
				// Post message to orderExecuted
				fmt.Println("Order executed !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!", avgPrice)
				positionEnteredChannel <- struct{}{}
				done = true
			} else {
				log.Fatalf("Cannot execute order %v", err)
			}
		}
	}
}

func exitAt3(testChannel chan os.Signal) {
	testRun, _ := strconv.ParseBool(os.Getenv("TEST_RUN"))
	if !testRun {
		for {
			if time.Now().Hour() >= 15 && time.Now().Minute() >= 35 {
				testChannel <- syscall.SIGTERM
				break
			}
			time.Sleep(180 * time.Second)
		}
	}
}

// Never trade, if candle height is more than 120
// Do not enter into trade, if loss is > 50K for the day

// check on reverse candle logic again...
// reduce log lines...

// After placing order in candleHt, why are we waiting forever.. if order never gets executed ????
// Should we just cancel it after 5 mins or if new candle is formed ???
//When under loss.. keep stop loss at 0.4% of entry price always....
// main.go... add killJob for V2
//Collect currentTicks of one day and use it for testing.. put all currentTicks into XL sheet, read it and send it to program one-by-one
// candle height buy/sell -- it buys immediately.. we should make sure that it comes in b/w start/end of current candle and then start the logic...
// Printing many times -- Sell order threshold start
// add starttick, ht to candle ht strategy...

// if buy is exited in profit.. then it might be coming down.. do not enter on the way down...
//enter only if it comes below endTick and starts moving up
