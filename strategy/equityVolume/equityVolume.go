package equityVolume

import (
	"encoding/csv"
	"fmt"
	"kite/order"
	myTicker "kite/ticker"
	"kite/types"
	"kite/util"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

type BNFDetails struct {
	Name      string
	Id        string
	ShortName string
}

var (
	kc            *kiteconnect.Client = nil
	envConf       *types.EnvConfig    = nil
	profitMetrics *types.Metrics

	finNiftyTickerIdNameMap map[uint32]string
	bnfDetailsList          []BNFDetails
	marketCap               string
)

var (
	startedTrading     map[uint32]string = make(map[uint32]string, 0)
	inFlightTrades     int               = 0
	TotalTradesAllowed int               = 4
	moneyAvailable     bool              = true
)

var (
	maxAmount float64 = 0.0
)

func init() {
	var err error
	envConf = util.GetEnvConfig()
	profitMetrics = new(types.Metrics)
	bnfDetailsList = make([]BNFDetails, 0)

	if !strings.HasPrefix(envConf.CFWSConf.TickerName, "EQUITY-VOLUME") {
		return
	}

	totalTradesStr := os.Getenv("TOTAL_TRADES")
	TotalTradesAllowed, err = strconv.Atoi(totalTradesStr)
	if err != nil {
		log.Fatalf("Error converting TOTAL_TRADES to int: %v", err)
	}

	str := os.Getenv("MAX_AMOUNT")
	maxAmount, err = strconv.ParseFloat(str, 64)
	if err != nil {
		log.Fatalf("Error converting MAX_AMOUNT to int: %v", err)
	}
	str = os.Getenv("MONEY_AVAILABLE")
	moneyAvailable, err = strconv.ParseBool(str)
	if err != nil {
		log.Fatalf("Error converting MAX_AMOUNT to int: %v", err)
	}

	fmt.Printf("Total trades %d, max amount %f and moneyAvailalbe %t\n", TotalTradesAllowed, maxAmount, moneyAvailable)
	setBNFDetails()
}

func ExecuteCandleHeightShareStrategy(client *kiteconnect.Client) {

	// Set all the required parameters
	kc = client

	// Give 10 seconds for currentTicks to update....
	time.Sleep(10 * time.Second)

	fmt.Println("Starting now to TRADE....................................................................")
	fmt.Println()

	//For bankNifty options -- todo..uncommet this later..
	go placeAndMonitorFinNiftyOptionsOrders()

}

func placeAndMonitorFinNiftyOptionsOrders() {

	fmt.Println("Entry: placeAndMonitorFinNiftyOptionsOrders")

	envConf.CFWSConf.LotSize = 30.0

	finNiftyTickerIdNameMap = make(map[uint32]string, 400)

	for _, obj := range bnfDetailsList {
		temp, err := strconv.Atoi(obj.Id)
		if err == nil {
			id := uint32(temp)
			finNiftyTickerIdNameMap[id] = obj.Name
		}
	}

	go startTrading("long")
}

func startTrading(position string) {

	fmt.Println("Start trading now...")

	myMarketMaker := MarketMakerSetup()

	//wait till 9:20
	testRun, _ := strconv.ParseBool(os.Getenv("TEST_RUN"))
	if !testRun {
		dt := time.Now()
		for dt.Hour() < 10 && dt.Minute() < 20 {
			time.Sleep(1 * time.Minute)
			dt = time.Now()
		}
	}

	//Initialize first time and wait for 60 seconds
	myMarketMaker.UpdateVelocityData(startedTrading)
	time.Sleep(60 * time.Second)

	for {
		//First updateVelocity and then start trading
		startTime := time.Now()
		myMarketMaker.UpdateVelocityData(startedTrading)
		if time.Now().Hour() >= 15 && time.Now().Minute() >= 1 && !testRun {
			// If time is 3:01 PM, do not enter new position...
			fmt.Println("Exit: placeAndMonitorFinNiftyOptionsOrders: ", time.Now().Hour(), time.Now().Minute())
			//Print analytics and exit..
			i := 0
			for _, tickerName := range startedTrading {
				i = i + 1
				tData := myMarketMaker.MarketTradeData[tickerName]
				fmt.Printf("Trade Number: %d. Details: %v\n", i, tData)
			}
			//	myMarketMaker.PrintDaysFinalSummary()
			fmt.Printf("Total trades: %d\n", len(startedTrading))
			fmt.Printf("Total inflight trades: %d\n", inFlightTrades)
			//	myMarketMaker.CreateReport()
			break
		}

		//Inspect all 500 currentTicks and then decide whether to trade or not.
		for optionsTickerId, optionsTickerName := range finNiftyTickerIdNameMap {

			currentOptionsTick := myTicker.GetFinNiftyCurrentTick(optionsTickerId)
			currentVolumeTraded, _ := myTicker.GetFinNiftyCurrentVolume(optionsTickerId)
			myMarketMaker.UpdateLiveData(optionsTickerName, currentOptionsTick, currentVolumeTraded)
			//fmt.Println("data....", optionsTickerName, currentOptionsTick, currentVolumeTraded)

			if startedTrading[optionsTickerId] != optionsTickerName {
				// if it is good to buy, then start a new thread...
				goodToTrade := myMarketMaker.isGoodToBuyBasedOnBuyIndex(optionsTickerName)
				if goodToTrade && inFlightTrades < TotalTradesAllowed {
					startedTrading[optionsTickerId] = optionsTickerName
					fmt.Printf("Start step:1. symbol:%s, tradeNumber:%v\n", optionsTickerName, inFlightTrades)
					go startTradingEquity(optionsTickerName, optionsTickerId, myMarketMaker, false)
				}
			}
		}
		//time.Sleep(30 * time.Second) //Take some rest and start next round...
		elapsed := int(time.Since(startTime).Seconds())
		//fmt.Println("sleeping for ", elapsed)
		if elapsed < 60 {
			time.Sleep(time.Duration(60-elapsed) * time.Second)
		}
		//fmt.Println("staring again...")
	}
}

func startTradingEquity(optionsTickerName string, tickerId uint32, myMarketMaker *marketMaker, bnfTest bool) {

	if !moneyAvailable {

		buyPrice := myTicker.GetFinNiftyCurrentTick(tickerId)
		lotSize := 0.0
		lotSize = float64(int(maxAmount / buyPrice)) //Buy for 10K
		if lotSize == 0 {
			bnfTest = true //I don't have enough money to buy even 1...
		}
		fmt.Printf("Setting lot size of %f for %s\n", lotSize, optionsTickerName)
		if !bnfTest {
			inFlightTrades = inFlightTrades + 1
		}

		// if money is not avaiable, assume it's executed and try to exit by using GTT
		fmt.Println("Money is not available. so placing GTT", optionsTickerName)
		PlaceSellOrderGTT(optionsTickerName, buyPrice*1.013, lotSize)
		return
	}

	exchange := "NSE"
	executed, exitPosition := false, false
	orderStatus := ""
	avgEntryPrice, avgExitPrice := 0.0, 0.0
	positionType := kiteconnect.TransactionTypeBuy //This is new.. we will always buy options..
	exitPositionType := kiteconnect.TransactionTypeSell

	fmt.Println("^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^yyyyyyyyyy")
	fmt.Printf("Place order start:%s\n", optionsTickerName)
	buyPrice := myTicker.GetFinNiftyCurrentTick(tickerId)
	lotSize := 0.0
	lotSize = float64(int(maxAmount / buyPrice)) //Buy for 10K
	if lotSize == 0 {
		bnfTest = true //I don't have enough money to buy even 1...
	}
	fmt.Printf("Setting lot size of %f for %s\n", lotSize, optionsTickerName)
	if !bnfTest {
		inFlightTrades = inFlightTrades + 1
	}
	myMarketMaker.PrepareToTrade(optionsTickerName, lotSize, buyPrice)

	util.MyPrintf("Start BUY BUY !!!!! sybmol: %s, qty: %f, buyPrice: %f, exchange: %s", optionsTickerName, lotSize, buyPrice, exchange)

	orderId, err := order.PlaceOrderSharesWithTagInNSE(kc, positionType, lotSize, buyPrice, optionsTickerName, bnfTest, "testing", exchange)
	if err != nil || orderId == "" {
		log.Fatalf("Cannot place buy options order %v", err)
	}
	if !bnfTest {
		fmt.Printf("Start monitoring entered BUY position status, orderId:%v, symbole:%s\n", orderId, optionsTickerName)
		go order.CancelOrder(kc, orderId, 300) // start a go routine to cancel order, if not executed in 120 seconds.
	}
	executed, avgEntryPrice, err, orderStatus = order.WaitForOrderToExecute(kc, orderId, bnfTest)
	if err != nil {
		log.Println("Executed !!!! ??????", executed)
		log.Fatalf("Error while checking for order status %v", err)
	}
	if bnfTest { //TODO - Instead of BNF Test, make use of GTT...
		avgEntryPrice = buyPrice
	}
	if orderStatus == kiteconnect.OrderStatusCancelled || orderStatus == kiteconnect.OrderStatusRejected {
		if !bnfTest {
			inFlightTrades = inFlightTrades - 1
		}
		if moneyAvailable {
			//If it got rejected even after money is available, then reset and try again...
			startedTrading[tickerId] = ""
			myMarketMaker.RemoveFromTrade(optionsTickerName)
		}
		fmt.Printf("At %v:%v order for %s with id %s is %s\n", time.Now().Hour(), time.Now().Minute(), orderId, orderStatus, optionsTickerName)
		if orderStatus == kiteconnect.OrderStatusRejected {
			if !moneyAvailable {
				avgEntryPrice = buyPrice
				// if money is not avaiable, assume it's executed and try to exit by using GTT
				fmt.Println("Money is not available. so placing GTT", optionsTickerName)
				PlaceSellOrderGTT(optionsTickerName, buyPrice*1.013, lotSize)
			}
			//fix below later..
			fmt.Println("order rejected..try later..TODO")
		}
	} else if executed {

		fmt.Printf("Buy position entered !!!!!!!!!!!!!!!! price:%v, symbol:%s, lotsize:%v, tradeNumber:%v\n", avgEntryPrice, optionsTickerName, lotSize, inFlightTrades)
		myMarketMaker.UpdateEntryTrade(optionsTickerName, avgEntryPrice)
		for !exitPosition {
			exitPrice := 0.0
			currentOptionsTick := myTicker.GetFinNiftyCurrentTick(tickerId)
			currentVolumeTraded, _ := myTicker.GetFinNiftyCurrentVolume(tickerId)
			myMarketMaker.UpdateLiveData(optionsTickerName, currentOptionsTick, currentVolumeTraded)
			goodToExit, exitPrice := myMarketMaker.isGoodToExitOptimized(optionsTickerName, LONG)
			if goodToExit {
				fmt.Println("Exit buy position: entryPrice, exitPrice", optionsTickerName, avgEntryPrice, exitPrice)
				myMarketMaker.PrepareToExit(optionsTickerName, exitPrice)
				exitPosition = true
			}

			if exitPosition {
				if !bnfTest {
					inFlightTrades = inFlightTrades - 1
				}
				orderId, err := order.PlaceOrderSharesWithTagInNSE(kc, exitPositionType, lotSize, exitPrice, optionsTickerName, bnfTest, "baseline", exchange)

				if err != nil || orderId == "" {
					fmt.Printf("Error: Cannot place exit SELL order %v for: %s\n", err, optionsTickerName)
					break
				}
				fmt.Println("Start monitoring exit - SELL position", optionsTickerName)
				executed, avgExitPrice, err, _ = order.WaitForOrderToExecute(kc, orderId, bnfTest)
				fmt.Println("Executed !!!! ??????", executed)
				if err != nil {
					log.Fatalf("Error while checking for order status %v", err)
				} else if !executed {
					fmt.Println("Not executed. will try again after few mins..")
					time.Sleep(1 * time.Minute)
					exitPosition = false
				} else {
					if bnfTest {
						avgExitPrice = exitPrice
					}
					fmt.Println("Exited at: ", avgExitPrice, time.Now().Hour(), time.Now().Minute())
					fmt.Printf("Profit is....Entry: %f, Exit: %f, Profit: %f \n", avgEntryPrice, avgExitPrice, lotSize*(avgExitPrice-avgEntryPrice))
					myMarketMaker.UpdateExitTrade(optionsTickerName, avgExitPrice)
				}
			}
			time.Sleep(2 * time.Second) //2 second rest before checking if we can exit...
		}

		// If we are here, then we have exited the position....
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		profitMetrics.BuyRounds = profitMetrics.BuyRounds + 1
		fmt.Println("%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%")
		fmt.Println("Done with buy and sell order number:", profitMetrics.BuyRounds)
	}
	fmt.Println("vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv")
}

func getTimeBasedProfitPerUnit(entryTime time.Time, entryPrice float64) float64 {
	multiplier := int(time.Since(entryTime).Minutes()) / 10
	exitPrice := 0.0
	if multiplier == 0 {
		// first 30 mins aim for max profit...
		exitPrice = (1.03 * entryPrice) + 1.5 //sell at 3% profit + 1.5Rs for brokerage
	} else if multiplier == 1 {
		// if crossed 30 mins.. then exit at entryPrice.
		exitPrice = entryPrice //after 30 mins..
	} else {
		// if crossed 60 mins.. then exit at 4% loss
		// if crossed 90 mins.. then exit at 6% loss.
		// if crossed 120 mins.. then exit at 8% loss and so on..
		lossPercentage := multiplier * 2 //from 60th min...
		exitPrice = entryPrice * float64(100-lossPercentage) / 100.0
	}
	return exitPrice
}

func getPointsBasedProfit(entryTick, entryPrice, currentTick float64, optionType string, entryTime time.Time) (float64, bool) {
	//If we are within 100 points of entryTick, then keep waiting...
	delta := 0.0
	stopLossFlag := false

	multiplier := int(time.Since(entryTime).Minutes()) / 60
	exitPrice := 0.0
	if multiplier == 0 {
		// first 60 mins aim for max profit...
		exitPrice = (1.03 * entryPrice) + 1.5 //sell at 3% profit + 1.5Rs for brokerage
	} else if multiplier >= 1 {
		// if crossed 60 mins.. then exit at entryPrice.
		exitPrice = entryPrice //after 60 mins..
	}

	if optionType == "call" {
		delta = entryTick - currentTick //Expect it to go up.. but if it's coming down by 100 points, then exit
	} else if optionType == "put" {
		delta = currentTick - entryTick
	}
	if delta > 125.0 {
		fmt.Println("STOP LOSS HIT :-( :-( )", currentTick, entryTick, optionType, delta)
		stopLossFlag = true
	}
	return exitPrice, stopLossFlag
}

func GetBNFDetails() []BNFDetails {
	return bnfDetailsList
}

func setBNFDetails() {

	//bnfDetailsList = append(bnfDetailsList, BNFDetails{"RELIANCE", "128083204", "call51600"})
	//bnfDetailsList = append(bnfDetailsList, BNFDetails{"BSE", "5013761", "put51600"})
	//bnfDetailsList = append(bnfDetailsList, BNFDetails{"HDFCBANK", "12957954", "call51700"})
	//bnfDetailsList = append(bnfDetailsList, BNFDetails{"TATAMOTORS", "884737", "put51700"})

	symbolList := make([]string, 400)

	//Read symbol to ID mapping CSV and generate the list...
	marketCap = os.Getenv("MARKET_CAP")
	if marketCap == "" {
		marketCap = "midcap"
	}

	//This file contains symbol of 400 midcap stocks and BSE Ids.
	// We will not use BSE Ids. Rather will red the instruments.csv to find the NSE Ids
	fileName := "c:/Guru/MidCap_400.csv"
	if marketCap == "midcap" {
		fileName = "c:/Guru/MidCap_400.csv"
	} else if marketCap == "smallcap" {
		fileName = "c:/Guru/SmallCap_250.csv"
	} else if marketCap == "microcap" {
		fileName = "c:/Guru/MicroCap_250.csv"
	}

	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Error opening file:", err)
		x := fmt.Sprintf("Error reading %s, %v", fileName, err)
		panic(x)
	}
	defer file.Close()

	// Create a new CSV reader
	reader := csv.NewReader(file)

	// Read all records from the file
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Error reading symbol to ID file:", err)
		fmt.Printf("Error is %v", err)
		panic(err)
	}

	for _, record := range records {
		//bnfDetailsList = append(bnfDetailsList, BNFDetails{record[1], record[2], "unusedField"})
		if marketCap == "midcap" {
			symbolList = append(symbolList, record[1])
		} else if marketCap == "smallcap" {
			symbolList = append(symbolList, record[0])
		} else if marketCap == "microcap" {
			symbolList = append(symbolList, record[0])
		}
	}

	fileName1 := "c:/Guru/instruments-nse.csv"
	file1, err1 := os.Open(fileName1)
	if err1 != nil {
		fmt.Println("Error opening file:", err1)
		panic("Error reading symbol to ID file")
	}
	defer file1.Close()

	// Create a new CSV reader
	reader1 := csv.NewReader(file1)

	// Read all records from the file
	records1, err1 := reader1.ReadAll()
	if err1 != nil {
		fmt.Println("Error reading temp nse ymbol to ID file:", err)
		panic("Error reading temp nse to ID file")
	}

	for _, record1 := range records1 {
		//fmt.Println("check if present, ", record1[2], record1[0])
		if present(record1[2], symbolList) {
			bnfDetailsList = append(bnfDetailsList, BNFDetails{record1[2], record1[0], "unusedField"})
		}
	}
	fmt.Printf("BNF Details List %v\n", bnfDetailsList)
}

func present(record string, records []string) bool {
	for _, rec := range records {
		if record == rec {
			return true
		}
	}
	return false
}

func ManageCommunication() {
	fmt.Println("Start INFY GTT creation for killing job .....")
	retryCount := 5
	var err error = nil
	var gttResp kiteconnect.GTTResponse
	i := 20.0
	for {
		i = i + 1
		// Place GTT
		gttResp, err = kc.PlaceGTT(kiteconnect.GTTParams{
			Tradingsymbol:   "INFY",
			Exchange:        "NSE",
			LastPrice:       800,
			TransactionType: kiteconnect.TransactionTypeBuy,
			Trigger: &kiteconnect.GTTSingleLegTrigger{
				TriggerParams: kiteconnect.TriggerParams{
					TriggerValue: 2,
					Quantity:     i,
					LimitPrice:   2,
				},
			},
		})
		if err != nil {
			fmt.Printf("error placing gtt: %v", err)
			retryCount = retryCount - 1
			if retryCount == 0 {
				fmt.Println("Error while placing GTT for INFY")
			}
		} else {
			fmt.Println("Gtt placed for INFY", gttResp.TriggerID)
			//break
			time.Sleep(20 * time.Minute)
		}
	}

}

func PlaceSellOrderGTT(symbol string, sellPrice, qty float64) {
	fmt.Printf("Start %s GTT creation for killing job .....\n", symbol)
	retryCount := 5
	var err error = nil
	var gttResp kiteconnect.GTTResponse
	// Place GTT
	gttResp, err = kc.PlaceGTT(kiteconnect.GTTParams{
		Tradingsymbol:   symbol,
		Exchange:        "NSE",
		LastPrice:       sellPrice / 2, //TODO is this needed
		TransactionType: kiteconnect.TransactionTypeSell,
		Trigger: &kiteconnect.GTTSingleLegTrigger{
			TriggerParams: kiteconnect.TriggerParams{
				TriggerValue: sellPrice,
				Quantity:     qty,
				LimitPrice:   sellPrice,
			},
		},
	})
	if err != nil {
		fmt.Printf("error placing gtt: %v", err)
		retryCount = retryCount - 1
		if retryCount == 0 {
			fmt.Printf("Error while placing GTT for %s\n", symbol)
		}
	} else {
		fmt.Printf("Gtt placed for %s with ID %v\n", symbol, gttResp.TriggerID)
	}
}
