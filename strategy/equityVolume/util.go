package equityVolume

import (
	"encoding/csv"
	"fmt"
	myTicker "kite/ticker"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	SYMBOL            = iota // 0
	SERIES                   // 1
	LTP                      // 2
	PERCENTAGE_CHANGE        // 3
	MARKET_CAP               // 4
	VOLUME                   // 5
	VALUE                    // 6
)

const (
	NUM_TIMES = 3
)

type position string

var mu sync.Mutex

const (
	LONG   position = "long"
	SHORT  position = "sell"
	CLOSED position = "closed"
)

type velocityData struct {
	symbol              string
	maxVelocity         float64
	minVelocity         float64
	currentVelocity     float64 //Calculated using currentTotalVolume-cumulativeVolume/(currentTime-lastUpdatedTime)
	avgVelocity         float64 //Calculated using startTime and cumulativeVolume
	startTime           time.Time
	startVolume         float64 //I need startVolume, bcs, I may start program at 10:00. I cannot use volme traded since 9:15 to calculate avg velocity
	lastUpdatedTime     time.Time
	cumulativeVolume    float64 //This is volume traded since startTime.
	startLTP            float64 //This is LTP since last volume/velocity was updated. Not since beginning like startVolume
	currentTrend        int
	totalBullRun        int
	totalBearRun        int
	totalTicks          int
	bannedForDay        bool
	topFiveVelocityAvg  float64
	topFiveVelocity     []float64
	tenDayHighestVolume float64
	avgCandleHt         float64

	meanVolume         float64
	meanVelocity       float64
	maxBuyIndex        float64
	topFiveBuyIndex    []float64
	topFiveBuyIndexAvg float64
	highestDayVolume   int //This is volume traded since startTime.

	previousCandleHt int
	currentCandleHt  int

	previousDayClosing float64
}

type historicalData struct {
	symbol        string
	dailyAvg      float64
	fiveDayAvg    float64
	tenDayAvg     float64
	highestVolume float64
	balanceVolume float64
	ltp           float64
	totalDays     int //N number of days
	totalVolume   float64
	highPrice     float64 //In last N days
	lowPrice      float64 //In last N days
}

type dailyData struct {
	symbol    string
	volume    float64
	ltp       float64
	dayNum    int
	direction int
}

type liveData struct {
	symbol  string
	volume  float64
	open    float64
	high    float64
	low     float64
	current float64
}

type tradeData struct {
	symbol             string
	position           position
	volume             float64
	unitEntryPrice     float64
	unitExitPrice      float64
	limitExitPrice     float64
	limitEntryPrice    float64
	exited             bool
	profit             float64
	percentageProfit   float64
	status             string //TODO add this...
	profitMaximization bool
	anchorPrice        float64
}

type marketMaker struct {
	MarketHistoricalData         map[string]*historicalData
	MarketDailyData              map[string][]*dailyData
	MarketLiveData               map[string]*liveData
	MarketTradeData              map[string]*tradeData
	MarketVelocityData           map[string]*velocityData
	MarketHistoricalVelocityData map[string]*velocityData
}

var myMarketMaker *marketMaker

func (m *marketMaker) PrepareToTrade(symbol string, volume float64, limitEntryPrice float64) {
	tData := m.MarketTradeData[symbol]
	if tData == nil {
		tData = new(tradeData)
		tData.symbol = symbol
		m.MarketTradeData[symbol] = tData
	}
	tData.volume = volume
	tData.limitEntryPrice = limitEntryPrice
	tData.position = LONG
	tData.status = "POSITION_ENTRY"
}

func (m *marketMaker) RemoveFromTrade(symbol string) {
	m.MarketTradeData[symbol] = nil
}

func (m *marketMaker) UpdateEntryTrade(symbol string, unitEntryPrice float64) {
	tradeData := m.MarketTradeData[symbol]
	tradeData.unitEntryPrice = unitEntryPrice
	tradeData.status = "POSITION_ENTERED"
}

func (m *marketMaker) PrepareToExit(symbol string, limitExitPrice float64) {
	tradeData := m.MarketTradeData[symbol]
	tradeData.limitExitPrice = limitExitPrice
	tradeData.status = "POSITION_EXIT"
}

func (m *marketMaker) UpdateExitTrade(symbol string, unitExitPrice float64) {
	tradeData := m.MarketTradeData[symbol]
	tradeData.unitExitPrice = unitExitPrice
	tradeData.exited = true
	tradeData.position = CLOSED
	profit := (tradeData.unitExitPrice - tradeData.unitEntryPrice) * tradeData.volume
	prec := (profit / (tradeData.unitEntryPrice * tradeData.volume)) * 100
	tradeData.profit = profit
	tradeData.percentageProfit = prec
	tradeData.status = "POSITION_EXITED"
}

func (m *marketMaker) UpdateVelocityData(startTrading map[uint32]string) {
	// fmt.Println("Update velocity data...")
	//TODO make use of closing price in the velocityData.csv

	//TODO later also calculate highest negative acceleration.. when volume goes up and price comes down
	mu.Lock()         // Lock the mutex before writing
	defer mu.Unlock() // Ensure the mutex is unlocked after writing
	for tickerId, symbol := range finNiftyTickerIdNameMap {
		if startTrading[tickerId] == symbol {
			//fmt.Println("skipping since it is already traded", symbol)
			continue
		}
		//avgCandleHt := 0
		// ltp := myTicker.GetFinNiftyCurrentTick(tickerId) - use for negative velocity
		currentVolumeTraded, timeTraded := myTicker.GetFinNiftyCurrentVolume(tickerId)
		currentTick := myTicker.GetFinNiftyCurrentTick(tickerId)
		vHistoricalData := m.MarketHistoricalVelocityData[symbol]
		vData := m.MarketVelocityData[symbol]
		/*if symbol == "ETHOSLTD" {
			fmt.Println("Inside update velocity for ETHOSLTD", currentVolumeTraded, time.Now().Minute(), time.Now().Second())
		}
		*/
		if currentVolumeTraded == 0 {
			continue
		}
		if vData == nil {
			vData = &velocityData{symbol: symbol}
			m.MarketVelocityData[symbol] = vData
			vData.cumulativeVolume = currentVolumeTraded
			vData.startVolume = currentVolumeTraded
			vData.startTime = time.Now()
			vData.lastUpdatedTime = timeTraded
			vData.maxVelocity = 0.0
			vData.currentVelocity = 0.0
			vData.startLTP = currentTick
			vData.currentTrend = 0
			vData.bannedForDay = false
			vData.previousCandleHt = 0
			vData.currentCandleHt = 0
			/*
				if vHistoricalData == nil {
					vData.topFiveVelocity = make([]float64, 5)
				} else {
					vData.topFiveVelocity = vHistoricalData.topFiveVelocity
					vData.topFiveVelocityAvg = vHistoricalData.topFiveVelocityAvg
				}
			*/
			//This is first time initialization. Cannot calculate velocity
		} else {
			//calculate velocity
			if vData.cumulativeVolume != currentVolumeTraded {
				vData.totalTicks = vData.totalTicks + 1
			}
			changeInVolume := currentVolumeTraded - vData.cumulativeVolume
			if changeInVolume > 0 {

				if vData.previousCandleHt == 0 {
					vData.previousCandleHt = int(changeInVolume)
				} else {
					vData.currentCandleHt = int(changeInVolume)
				}

				timeTaken := timeTraded.Sub(vData.lastUpdatedTime).Minutes()
				if timeTaken == 0 {
					timeTaken = timeTraded.Sub(vData.lastUpdatedTime).Seconds() / 60
				}
				velocity := math.Round(changeInVolume / timeTaken)

				//Update the velocity
				vData.currentVelocity = velocity

				vData.lastUpdatedTime = timeTraded
				/*if symbol == "ETHOSLTD" {
					fmt.Println("velocity calculation...oh my god", currentTick, timeTaken, currentVolumeTraded, vData.cumulativeVolume, velocity, time.Now().Minute(), time.Now().Second())
				}
				*/
				if vData.maxVelocity < velocity {
					if currentTick > vData.startLTP {
						//It's a green candle...use this data...
						vData.maxVelocity = velocity
					}
					//If price is less.. it's a big red candle. Stop trading for the day...
					if vData.startLTP > currentTick {
						if vHistoricalData != nil && vHistoricalData.maxVelocity < vData.maxVelocity {
							//fmt.Printf("Can be banned for the day !!! %s %v:%v\n", symbol, time.Now().Hour(), time.Now().Minute())
							//TODO - important add this later...
							//vData.bannedForDay = true
						}
					}
				}
				if vData.startLTP < currentTick { // price is increasing
					vData.currentTrend = 1
				} else {
					vData.currentTrend = -1
				}
				//Update our data
				vData.cumulativeVolume = currentVolumeTraded
				vData.startLTP = currentTick
				if vData.currentCandleHt > 0 {
					vData.previousCandleHt = vData.currentCandleHt
				}
				vData.currentCandleHt = 0
			}

		}
	}
}

func mergeData(t *dailyData, h *historicalData) {
	//Confirm symbols are same
	if t.symbol != h.symbol {
		// TODO throw error
		panic("Invalid merge operation")
	}
	//Take the latest dailyData.
	if t.dayNum > h.totalDays {
		h.ltp = t.ltp
	}

	h.totalDays = h.totalDays + 1
	h.totalVolume = h.totalVolume + t.volume
	h.balanceVolume = h.balanceVolume + (t.volume)*float64(t.direction)
	//fmt.Println("Aaa", t.volume)
	h.dailyAvg = roundOff(h.totalVolume / float64(h.totalDays))

	//Averages
	if h.totalDays == 5 {
		h.fiveDayAvg = h.dailyAvg
	} else if h.totalDays == 10 {
		h.tenDayAvg = h.dailyAvg
	}
	//High price
	if h.highPrice < t.ltp {
		h.highPrice = t.ltp
	}
	//Low price
	if h.lowPrice > t.ltp {
		h.lowPrice = t.ltp
	}

	//highest volume
	if h.highestVolume < t.volume {
		h.highestVolume = t.volume
	}
	//fmt.Println("Daily avg", h.dailyAvg)

}

func roundOff(x float64) float64 {
	return math.Round(x*100) / 100
}

func newMarketMaker() *marketMaker {
	m := new(marketMaker)
	if m.MarketDailyData == nil {
		m.MarketDailyData = make(map[string][]*dailyData, 400)
	}
	if m.MarketHistoricalData == nil {
		m.MarketHistoricalData = make(map[string]*historicalData, 400)
	}
	if m.MarketLiveData == nil {
		m.MarketLiveData = make(map[string]*liveData, 400)
	}
	if m.MarketTradeData == nil {
		m.MarketTradeData = make(map[string]*tradeData, 400)
	}
	if m.MarketVelocityData == nil {
		m.MarketVelocityData = make(map[string]*velocityData, 400)
	}
	if m.MarketHistoricalVelocityData == nil {
		m.MarketHistoricalVelocityData = make(map[string]*velocityData, 400)
	}
	return m
}

func (m *marketMaker) init(dayNum int, records [][]string) {
	for _, record := range records {
		data := getDailyData(dayNum, record)
		m.MarketDailyData[data.symbol] = append(m.MarketDailyData[data.symbol], data)
	}
}

func (m *marketMaker) MergeVelocityData() {

	mu.Lock()         // Lock the mutex before writing
	defer mu.Unlock() // Ensure the mutex is unlocked after writing

	//symbol, maxVelocity, topFiveVelocity[0],topFiveVelocityAvg, highestDayVolume
	fileName := "c:/Guru/historicalVelocity.csv"
	file, err := os.Open(fileName)
	fmt.Println("Reding file", fileName)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Create a new CSV reader
	reader := csv.NewReader(file)

	// Read all records from the file
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	for _, record := range records {
		if len(record) >= 4 {
			symbol := record[0]
			vHData := m.MarketHistoricalVelocityData[symbol]

			if vHData == nil {
				vHData = new(velocityData)
				vHData.symbol = symbol
				vHData.topFiveVelocity = make([]float64, 5)
				m.MarketHistoricalVelocityData[symbol] = vHData
			}

			//symbol,topFiveVelocityAvg,topFiveBuyIndexAvg,closingPrice

			velocityAvg, _ := strconv.ParseFloat(record[1], 64)
			buyIndexAvg, _ := strconv.ParseFloat(record[2], 64)
			previousDayClosing, _ := strconv.ParseFloat(record[3], 64)

			vHData.topFiveVelocityAvg = velocityAvg
			vHData.topFiveBuyIndexAvg = buyIndexAvg
			vHData.previousDayClosing = previousDayClosing
			m.MarketHistoricalVelocityData[symbol] = vHData
		} else {
			fmt.Printf("this record in file is incomplete %v\n", record)
		}
	}
	fmt.Println("Final historical velocity data to use for the day")
	fmt.Println("&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&")
	fmt.Printf("vHData.symbol, vHData.topFiveVelocityAvg, vHData.topFiveBuyIndexAvg")
	for _, vHData := range m.MarketHistoricalVelocityData {
		fmt.Printf("%s %.2f %.2f\n", vHData.symbol, vHData.topFiveVelocityAvg, vHData.topFiveBuyIndexAvg)
	}
	fmt.Println("&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&")

}

/*
func (m *marketMaker) UpdatePriceVelocityData() {
	// fmt.Println("Update price velocity data...")

	//TODO later also calculate highest negative acceleration.. when volume goes up and price comes down
	mu1.Lock()         // Lock the mutex before writing
	defer mu1.Unlock() // Ensure the mutex is unlocked after writing
	for tickerId, symbol := range finNiftyTickerIdNameMap {

		// ltp := myTicker.GetFinNiftyCurrentTick(tickerId) - use for negative velocity
		currentTick := myTicker.GetFinNiftyCurrentTick(tickerId)
		pvData := m.MarketPriceVelocityData[symbol]
		if pvData == nil {
			pvData = &priceVelocityData{symbol: symbol}
			m.MarketPriceVelocityData[symbol] = pvData
			pvData.startTime = time.Now()
			pvData.lastUpdatedTime = time.Now()
			pvData.maxVelocity = 0.0
			pvData.currentVelocity = 0.0
			pvData.LTP = currentTick
			pvData.startPrice = currentTick
			pvData.currentTrend = 0
			pvData.minVelocity = 999999999.0 //some large number so that it gets reset again.....
			//This is first time initialization. Cannot calculate velocity
		} else {
			//calculate velocity
			changeInPrice := ((currentTick - pvData.LTP) / pvData.LTP) * 1000.0 //Velocity data was very small..So multiplying by 1000 to make it more readable..
			if changeInPrice > 0 {
				timeTaken := time.Since(pvData.lastUpdatedTime).Minutes()
				velocity := math.Round(changeInPrice / timeTaken)
				//Update the data
				avgVelocity := math.Round(((currentTick - pvData.startPrice) / pvData.startPrice) / time.Since(pvData.startTime).Minutes())
				pvData.currentVelocity = velocity
				pvData.avgVelocity = avgVelocity
				pvData.lastUpdatedTime = time.Now()
				if pvData.maxVelocity < velocity {
					pvData.maxVelocity = velocity
				}
				if pvData.minVelocity > velocity {
					pvData.minVelocity = velocity
				}
				if pvData.LTP < currentTick { // price is increasing
					pvData.currentTrend = 1
				} else {
					pvData.currentTrend = -1
				}
				pvData.LTP = currentTick
			}

		}
	}
}
*/

/*
func updateTopFiveVelocity(velocity float64, topFive []float64) []float64 {
	contains := false
	for _, v := range topFive {
		if v == velocity {
			contains = true
			break
		}
	}
	if !contains {
		if velocity > topFive[0] {
			topFive[0] = velocity
			slices.Sort(topFive)
		}
	}
	return topFive
}
*/

/*
func (m *marketMaker) MergePriceVelocityData() {

	mu1.Lock()         // Lock the mutex before writing
	defer mu1.Unlock() // Ensure the mutex is unlocked after writing

	for i := 1; i < 6; i++ {
		fileName := fmt.Sprintf("c:/Guru/priceVelocityData_%d.csv", i)
		file, err := os.Open(fileName)
		fmt.Println("Reding file", fileName)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
		defer file.Close()

		// Create a new CSV reader
		reader := csv.NewReader(file)

		// Read all records from the file
		records, err := reader.ReadAll()
		if err != nil {
			fmt.Println("Error reading file:", err)
			return
		}

		for _, record := range records {
			if len(record) >= 4 {
				symbol := record[0]
				maxV, e1 := strconv.ParseFloat(record[1], 64)
				minV, e2 := strconv.ParseFloat(record[2], 64)
				avgV, e3 := strconv.ParseFloat(record[3], 64)
				if e1 != nil {
					fmt.Println(e1, symbol, record)
					panic(e1)
				}
				if e2 != nil {
					fmt.Println(e2, symbol, record)
					panic(e2)
				}
				if e3 != nil {
					fmt.Println(e3, symbol, record)
					panic(e3)
				}
				pvData := m.MarketHistoricalPriceVelocityData[symbol]
				if pvData == nil {
					pvData = new(priceVelocityData)
					pvData.symbol = symbol
					m.MarketHistoricalPriceVelocityData[symbol] = pvData
				}
				if pvData.maxVelocity < maxV {
					pvData.maxVelocity = maxV
				}
				if pvData.minVelocity > minV {
					pvData.minVelocity = minV
				}
				newAvg := (avgV + pvData.avgVelocity) / 2
				pvData.avgVelocity = newAvg
				m.MarketHistoricalPriceVelocityData[symbol] = pvData
			} else {
				fmt.Printf("this record in file %d is incomplete %v\n", i, record)
			}
		}
	}

	fmt.Println("&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&")

	for _, pvData := range m.MarketHistoricalPriceVelocityData {
		fmt.Println(pvData.symbol, pvData.maxVelocity, pvData.minVelocity)
	}

}
*/

func (m *marketMaker) generateHistoricalData() {
	for sym, dDataList := range m.MarketDailyData {
		hData := m.MarketHistoricalData[sym]
		if hData == nil {
			hData = new(historicalData)
			hData.symbol = sym
			m.MarketHistoricalData[sym] = hData
		}
		for _, dData := range dDataList {
			//fmt.Println("aaaaa")
			mergeData(dData, hData)
		}
	}
}

func (m *marketMaker) CreateReport() {
	// Open the CSV file in append mode, creating it if it doesn't exist
	file, err := os.OpenFile("c:/Guru/StockSummaryReport.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer file.Close()

	// Create a CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Data to write or append to the CSV file
	records := make([][]string, 0)
	rowVal := []string{"Symbol", "Qty", "Entry Price", "Exit Price", "Profit", "% Profit", "Exited"}
	records = append(records, rowVal)
	for _, tData := range m.MarketTradeData {
		if tData != nil {
			rowVal = []string{tData.symbol, getStringVal(tData.volume), getStringVal(tData.unitEntryPrice), getStringVal(tData.unitExitPrice), getStringVal(tData.profit), getStringVal(tData.percentageProfit), strconv.FormatBool(tData.exited)}
			records = append(records, rowVal)
		}
	}

	// Write each record to the CSV file
	for _, record := range records {
		if err := writer.Write(record); err != nil {
			fmt.Println("Error writing record to CSV:", err)
			return
		}
	}

	// m.writeVelocityDataToFile()
	// m.writePriceVelocityDataToFile()
}

func getStringVal(val float64) string {
	return strconv.FormatFloat(val, 'f', 2, 64)
}

func getDailyData(dayNum int, record []string) *dailyData {
	data := new(dailyData)
	data.symbol = record[0]
	data.dayNum = dayNum
	val, err := strconv.ParseFloat(record[LTP], 32)
	if err == nil {
		data.ltp = roundOff(val)
	}
	val, err = strconv.ParseFloat(record[VOLUME], 32)
	if err == nil {
		data.volume = roundOff(val * 100000)
	}
	val, err = strconv.ParseFloat(record[PERCENTAGE_CHANGE], 32)

	if err == nil {
		if val < 0 {
			data.direction = -1
		} else {
			data.direction = 1
		}
	}
	return data
}

func (m *marketMaker) isGoodToExit(symbol string, positionType position) (bool, float64) {
	mu.Lock()         // Lock the mutex before writing
	defer mu.Unlock() // Ensure the mutex is unlocked after writing
	lData := m.MarketLiveData[symbol]
	tData := m.MarketTradeData[symbol]
	//vHData := m.MarketHistoricalVelocityData[symbol]
	//vData := m.MarketVelocityData[symbol]
	multiplier := 1.013

	if positionType == LONG {
		if lData.current > tData.unitEntryPrice*multiplier {
			// tData.limitExitPrice = lData.current
			return true, lData.current //if it is 1.13% profit, then exit
		}
	}
	dt := time.Now()
	if positionType == LONG && dt.Hour() >= 15 && dt.Minute() >= 15 {
		if lData.current > tData.unitEntryPrice {
			fmt.Printf("Exiting at 3:00 PM for %s\n", symbol)
			return true, lData.current
		}
	}
	return false, tData.unitEntryPrice * multiplier
}

func (m *marketMaker) isGoodToExitOptimized(symbol string, positionType position) (bool, float64) {
	mu.Lock()         // Lock the mutex before writing
	defer mu.Unlock() // Ensure the mutex is unlocked after writing
	lData := m.MarketLiveData[symbol]
	tData := m.MarketTradeData[symbol]
	defaultMultiplier := 1.015

	if !tData.profitMaximization {
		if lData.current > tData.unitEntryPrice*defaultMultiplier {
			tData.profitMaximization = true
			tData.anchorPrice = 1.5 //% to exit..
			//starting to do profit maximization. Return false to exit
			return false, tData.unitEntryPrice * defaultMultiplier
		}
	} else {
		//if profitMaximization started.. price crossed 1.3% of entry at least once...
		profitMultiplier := ((tData.anchorPrice + 0.5) / 100) + 1.0
		slUpperCircuit := ((tData.anchorPrice - 0.3) / 100) + 1.0
		slLowerCircuit := ((tData.anchorPrice - 0.5) / 100) + 1.0
		if lData.current > tData.unitEntryPrice*profitMultiplier {
			//Move the profit price further...
			tData.anchorPrice = tData.anchorPrice + 0.5
			fmt.Printf("increasing anchor price for : %s to %f\n", tData.symbol, tData.anchorPrice)
			return false, tData.unitEntryPrice * defaultMultiplier
		} else if lData.current < tData.unitEntryPrice*slUpperCircuit && lData.current > tData.unitEntryPrice*slLowerCircuit {
			fmt.Printf("Hitting StopLoss for %s with anchor price: %f and current price is: %f\n", tData.symbol, tData.anchorPrice, lData.current)
			return true, lData.current
		}
	}
	dt := time.Now()
	//postionType is unnecessary here.. it will be always LONG
	if positionType == LONG && dt.Hour() >= 15 && dt.Minute() >= 15 {
		if lData.current > tData.unitEntryPrice*1.003 {
			fmt.Printf("Exiting at 3:00 PM for %s\n", symbol)
			return true, lData.current
		}
	}
	return false, tData.unitEntryPrice * defaultMultiplier
}

// TODO.. Also update open/high/low/close later
func (m *marketMaker) UpdateLiveData(symbol string, ltp, volume float64) {
	mu.Lock()         // Lock the mutex before writing
	defer mu.Unlock() // Ensure the mutex is unlocked after writing
	lData := new(liveData)
	lData.symbol = symbol
	lData.current = ltp
	lData.volume = volume
	m.MarketLiveData[symbol] = lData
}

/*
1. current price should be > ltp
2. volume should be > highestVolume
3. volume should be > fiveDayAvg and dailyAvg
*/
func (m *marketMaker) isGoodToBuyBasedOnVolume(symbol string) bool {
	lData := m.MarketLiveData[symbol]
	hData := m.MarketHistoricalData[symbol]
	if lData == nil || hData == nil {
		fmt.Printf("Live data is nil for symbol:%s and lData:%v\n", symbol, lData)
		return false
	}
	if lData.current > hData.ltp*1.02 { //Greater than 2%
		//fmt.Println("is more than 2%", symbol, hData.dailyAvg, hData.highestVolume, lData.volume)
		if m.isBreakingVolume(lData) {
			fmt.Println("is more than 2% BUY ", symbol, hData.dailyAvg, hData.highestVolume, lData.volume, lData.current, hData.ltp)
			return true
		}
	}
	return false
}

//TODO... as soon as we calculate the maxVelocity breaking the historical maxVelocit, add it to a list of stocks to trade
// Loop through the list of stocks to trade and pick it up and trade it immediately.. this change is needed ..****** imp TODO

// This is based on velocity
func (m *marketMaker) isGoodToBuy(symbol string) bool {
	mu.Lock()         // Lock the mutex before writing
	defer mu.Unlock() // Ensure the mutex is unlocked after writing
	vHistoricalData := m.MarketHistoricalVelocityData[symbol]
	vData := m.MarketVelocityData[symbol]
	if vData != nil && vData.bannedForDay {
		fmt.Printf("%s is banned for the day\n", symbol)
		return false
	}
	if vHistoricalData == nil {
		//fmt.Printf("historical v data is nil for symbol:%s\n", symbol)
		return false
	}
	if vData == nil {
		//fmt.Printf("v data is nil for symbol:%s\n", symbol)
		return false
	}
	//use the below one.. TODO
	if vData.currentVelocity > vHistoricalData.topFiveVelocityAvg { //&& vData.avgCandleHt > vHistoricalData.avgCandleHt {
		//if vData.currentVelocity > vHistoricalData.maxVelocity {
		fmt.Printf("Velocity is more %s, Current:%f, Max:%f, Avg:%f\n", symbol, vData.currentVelocity, vHistoricalData.maxVelocity, vHistoricalData.topFiveVelocityAvg)
		if vData.currentTrend <= 0 {
			fmt.Printf("But current trend is negative for %s\n", symbol)
			return false
		}
		lData := m.MarketLiveData[symbol]
		hData := m.MarketHistoricalData[symbol]
		if lData == nil || hData == nil {
			fmt.Printf("Live data is nil for symbol:%s and lData:%v\n", symbol, lData)
			return false
		}
		if lData.current > hData.ltp*1.01 { //Greater than 1%. chagne this to use avg traded price..
			return true
		}
	}
	return false
}

/*
1. current price should be < ltp
2. volume should be < highestVolume
3. volume should be < fiveDayAvg and dailyAvg
*/
func (m *marketMaker) isGoodToSell(symbol string) bool {
	lData := m.MarketLiveData[symbol]
	hData := m.MarketHistoricalData[symbol]
	if lData.current < hData.ltp {
		if m.isBreakingVolume(lData) {
			fmt.Println("Time to sell....")
			return true
		}
	}
	return false
}

func (m *marketMaker) isBreakingVolume(lData *liveData) bool {
	symbol := lData.symbol
	hData := m.MarketHistoricalData[symbol]
	if lData.volume > hData.highestVolume/2 && lData.volume > hData.dailyAvg && lData.volume > hData.fiveDayAvg {
		if hData.balanceVolume+lData.volume > 0 {
			return true
		} else {
			fmt.Printf("The balance volume is negative %s, %f, %f, %f", symbol, hData.totalVolume, hData.balanceVolume, lData.volume)
		}
	}
	return false
}

func (m marketMaker) String() string {
	var sb strings.Builder
	fmt.Println("1")
	sb.WriteString("Market Maker Data:\n")
	for _, data := range m.MarketHistoricalData {
		if data.symbol == "RELIANCE" {
			fmt.Println("111")
			sb.WriteString(data.String() + "\n")
		}
	}
	return sb.String()
}

func (hd historicalData) String() string {
	return fmt.Sprintf("Records are : ", hd.symbol, hd.dailyAvg, hd.fiveDayAvg, hd.ltp, hd.highPrice, hd.highestVolume)
}

func (td tradeData) String() string {

	ltp := 0.0
	for tickerId, symbol := range finNiftyTickerIdNameMap {
		if td.symbol == symbol {
			ltp = myTicker.GetFinNiftyCurrentTick(tickerId)
			break
		}
	}

	dayEndProfit := td.volume * (ltp - td.unitEntryPrice)
	return fmt.Sprintf("Symbol: %s, Entry Price: %f, Exit Price: %f, Profit: %f, Percentage Profit: %f LTP: %f Day End Profit: %f",
		td.symbol, td.unitEntryPrice, td.unitExitPrice, td.profit, td.percentageProfit, ltp, dayEndProfit)
}

func (m marketMaker) PrintDaysFinalSummary() {
	closedTrades := 0
	totalProfit := 0.0
	openProfit := 0.0
	openTrades := 0
	i := 0
	for _, td := range m.MarketTradeData {
		if td != nil {
			i = i + 1
			ltp := 0.0
			for tickerId, symbol := range finNiftyTickerIdNameMap {
				if td.symbol == symbol {
					ltp = myTicker.GetFinNiftyCurrentTick(tickerId)
					break
				}
			}
			if td.status == "POSITION_EXITED" {
				fmt.Printf("Closed Trade: %d, Symbol: %s, Entry Price: %f, Exit Price: %f, Profit: %f, Percentage Profit: %f, Closing Price: %f\n", i, td.symbol, td.unitEntryPrice, td.unitExitPrice, td.profit, td.percentageProfit, ltp)
				totalProfit = totalProfit + td.profit
				closedTrades = closedTrades + 1
			} else {
				//m.UpdateExitTrade(td.symbol, ltp)
				fmt.Printf("Open Trade: %d, Symbol: %s, Entry Price: %f, Exit Price: %f, Profit: %f, Percentage Profit: %f, Closing Price: %f\n", i, td.symbol, td.unitEntryPrice, td.unitExitPrice, td.profit, td.percentageProfit, ltp)
				openTrades = openTrades + 1
				profit := td.volume * (ltp - td.unitEntryPrice)
				openProfit = openProfit + profit
			}
		}
	}
	fmt.Printf("Total Closed Profit : %f and total trades: %d\n", totalProfit, closedTrades)
	fmt.Printf("Total Open Profit : %f and total trades: %d\n", openProfit, openTrades)

}

func (m marketMaker) PlaceGTTForPendingPositions() {
	for _, td := range m.MarketTradeData {
		if td != nil {
			if td.status != "POSITION_EXITED" {
				fmt.Printf("Placing GTT Symbol: %s, Entry Price: %f,\n", td.symbol, td.unitEntryPrice)
				price := td.unitEntryPrice * 1.02
				price = math.Round(price*10) / 10
				PlaceSellOrderGTT(td.symbol, price, td.volume)
			}
		}
	}

	for _, td := range m.MarketTradeData {
		if td != nil {
			if td.status != "POSITION_EXITED" {
				//Later add data about when it entered.. TODO MUST
				fmt.Printf("%s 0 %.2f\n", td.symbol, td.unitEntryPrice)
			}
		}
	}

	for _, td := range m.MarketTradeData {
		if td != nil {
			if td.status == "POSITION_EXITED" {
				//Later add data about when it entered.. TODO MUST
				fmt.Printf("%s 1 %.2f\n", td.symbol, td.unitEntryPrice)
			}
		}
	}
}

func GetMarketMakerSetup() *marketMaker {
	return myMarketMaker
}

func MarketMakerSetup() *marketMaker {
	fmt.Println("OM")

	myMarketMaker = newMarketMaker()
	// https://www.nseindia.com/market-data/stocks-traded
	// https://www.nseindia.com/market-data/volume-gainers-spurts
	// https://www.nseindia.com/market-data/most-active-equities
	// https://www.nseindia.com/market-data/live-equity-market
	// Open the CSV file
	//for i := 1; i < 17; i++ {
	fileName := "c:/Guru/StocksTraded.csv"
	file, err := os.Open(fileName)
	fmt.Println("Reding file", fileName)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return nil
	}
	defer file.Close()

	// Create a new CSV reader
	reader := csv.NewReader(file)

	// Read all records from the file
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Error reading file:", err)
		return nil
	}
	myMarketMaker.init(1, records)
	//}

	//Finally generateHistoricalData
	myMarketMaker.generateHistoricalData()
	fmt.Printf("%v", myMarketMaker)
	myMarketMaker.MergeVelocityData()
	//enable this once we have the csv available...
	//myMarketMaker.MergePriceVelocityData()
	return myMarketMaker
}

func (m *marketMaker) isGoodToBuyBasedOnBuyIndex(symbol string) bool {
	mu.Lock()         // Lock the mutex before writing
	defer mu.Unlock() // Ensure the mutex is unlocked after writing
	vHistoricalData := m.MarketHistoricalVelocityData[symbol]
	vData := m.MarketVelocityData[symbol]
	if vData != nil && vData.bannedForDay {
		fmt.Printf("%s is banned for the day\n", symbol)
		return false
	}
	if vHistoricalData == nil {
		//fmt.Printf("historical v data is nil for symbol:%s\n", symbol)
		return false
	}
	if vData == nil {
		//fmt.Printf("v data is nil for symbol:%s\n", symbol)
		return false
	}
	//use the below one.. TODO
	if vData.currentVelocity > vHistoricalData.topFiveVelocityAvg {
		fmt.Printf("Velocity is more %s, Current:%f, Max:%f, Avg:%f\n", symbol, vData.currentVelocity, vHistoricalData.maxVelocity, vHistoricalData.topFiveVelocityAvg)
		if vData.currentTrend <= 0 {
			fmt.Printf("But current trend is negative for %s\n", symbol)
			return false
		}
		lData := m.MarketLiveData[symbol]
		hData := m.MarketHistoricalData[symbol]
		if lData == nil || hData == nil {
			fmt.Printf("Live data is nil for symbol:%s and lData:%v\n", symbol, lData)
			return false
		}
		if lData.current > hData.ltp*1.01 { //Greater than 1%. change this to use avg traded price..
			return true
		}
	}
	return false
}

/*
func (m *marketMaker) writeVelocityDataToFile() {
	// Define the file name
	fileName := "c:/Guru/velocityData.csv"

	// Check if file exists
	var file *os.File
	if _, err := os.Stat(fileName); errors.Is(err, os.ErrNotExist) {
		// File doesn't exist, so create it
		file, err = os.Create(fileName)
		if err != nil {
			fmt.Printf("failed to create file: %v", err)
		}
	} else {
		// File exists, open for reading and updating
		file, err = os.OpenFile(fileName, os.O_RDWR, 0644)
		if err != nil {
			fmt.Printf("failed to open file: %v", err)
		}
	}

	defer file.Close()

	// Write updated data back to file
	//file.Truncate(0)
	//file.Seek(0, 0)

	writer := csv.NewWriter(file)
	for symbol, vData := range m.MarketVelocityData {
		record := make([]string, 4)
		record[0] = symbol
		record[0] = symbol
		record[1] = strconv.FormatFloat(vData.maxVelocity, 'f', 2, 64)
		record[2] = strconv.FormatFloat(vData.minVelocity, 'f', 2, 64)
		record[3] = strconv.FormatFloat(vData.avgVelocity, 'f', 2, 64)
		if err := writer.Write(record); err != nil {
			fmt.Println("Error writing velcoity data record to CSV:", err)
			return
		}
	}

	fmt.Println("velocity data CSV file updated successfully:", len(m.MarketVelocityData))
}

// Function to write headers to a new CSV file
func writeHeaders(file *os.File) {
	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{"Symbol", "Min Velocity", "Average Velocity", "Max Velocity"}
	if err := writer.Write(headers); err != nil {
		fmt.Printf("failed to write headers: %v", err)
	}
}

*/

/*

func (m *marketMaker) writeVelocityDataToFile_backup() {
	// Define the file name
	fileName := "c:/Guru/velocityData.csv"

	// Check if file exists
	var file *os.File
	var err error
	if _, err := os.Stat(fileName); errors.Is(err, os.ErrNotExist) {
		// File doesn't exist, so create it
		file, err = os.Create(fileName)
		if err != nil {
			fmt.Printf("failed to create file: %v", err)
		}
		// Write headers if creating a new file
		//	writeHeaders(file) //TODO later
	} else {
		// File exists, open for reading and updating
		file, err = os.OpenFile(fileName, os.O_RDWR, 0644)
		if err != nil {
			fmt.Printf("failed to open file: %v", err)
		}
	}

	defer file.Close()

	// Read existing data into a map for easy replacement
	records, err := readCSVToMap(file)
	if err != nil {
		fmt.Printf("failed to read CSV data: %v", err)
	}

	updateData(records, m.MarketVelocityData)

	// Write updated data back to file
	file.Truncate(0)
	file.Seek(0, 0)
	writeUpdatedData(file, m.MarketVelocityData)
	fmt.Println("CSV file updated successfully")
}


// Function to read CSV data into a map with ID as the key
func readCSVToMap(file *os.File) ([][]string, error) {
	reader := csv.NewReader(file)
	// Read all records from the file
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	return records, nil
}

func updateData(fileRecords [][]string, vDatas map[string]*velocityData) {
	//Do not yet write headers.. we have to skip it or handle parse float error...
	for sybmol, vData := range vDatas {
		minVelocity := vData.minVelocity
		avgVelocity := vData.avgVelocity
		maxVelocity := vData.maxVelocity
		for _, fileRecord := range fileRecords {
			if fileRecord[0] == sybmol {
				min, _ := strconv.ParseFloat(fileRecord[1], 64)
				avg, _ := strconv.ParseFloat(fileRecord[2], 64)
				max, _ := strconv.ParseFloat(fileRecord[3], 64)
				if minVelocity > min {
					vData.minVelocity = min
				}
				if maxVelocity < max {
					vData.maxVelocity = max
				}
				//TODO this is just approximation. We have to keep total volume and total time elapsed and keep calculating avg daily.
				finalAvg := (avgVelocity + avg) / 2
				vData.avgVelocity = finalAvg
			}
		}
		//TODO .. any missing records prsent in file should be added back to myRecords
	}
}

// Function to write updated records back to the CSV file
func writeUpdatedData(file *os.File, vData map[string]*velocityData) {
	writer := csv.NewWriter(file)
	defer writer.Flush()

	for symbol, vData := range vData {
		record := make([]string, 4)
		record[0] = symbol
		record[1] = strconv.FormatFloat(vData.minVelocity, 'f', 2, 64)
		record[2] = strconv.FormatFloat(vData.avgVelocity, 'f', 2, 64)
		record[3] = strconv.FormatFloat(vData.maxVelocity, 'f', 2, 64)
		if err := writer.Write(record); err != nil {
			fmt.Println("Error writing record to CSV:", err)
			return
		}
	}
}
*/

/*
func (m *marketMaker) MergeVelocityData1() {

	mu.Lock()         // Lock the mutex before writing
	defer mu.Unlock() // Ensure the mutex is unlocked after writing

	for i := 1; i < 15; i++ {
		fileName := fmt.Sprintf("c:/Guru/velocity_%d.csv", i)
		file, err := os.Open(fileName)
		fmt.Println("Reding file", fileName)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
		defer file.Close()

		// Create a new CSV reader
		reader := csv.NewReader(file)

		// Read all records from the file
		records, err := reader.ReadAll()
		if err != nil {
			fmt.Println("Error reading file:", err)
			return
		}

		for _, record := range records {
			if len(record) >= 4 {
				symbol := record[0]
				maxV, e1 := strconv.ParseFloat(record[1], 64)
				minV, e2 := strconv.ParseFloat(record[2], 64)
				avgV, e3 := strconv.ParseFloat(record[3], 64)
				if e1 != nil {
					fmt.Println(e1, symbol, record)
					panic(e1)
				}
				if e2 != nil {
					fmt.Println(e2, symbol, record)
					panic(e2)
				}
				if e3 != nil {
					fmt.Println(e3, symbol, record)
					panic(e3)
				}
				vHData := m.MarketHistoricalVelocityData[symbol]
				if vHData == nil {
					vHData = new(velocityData)
					vHData.symbol = symbol
					m.MarketHistoricalVelocityData[symbol] = vHData
				}
				if vHData.maxVelocity < maxV {
					vHData.maxVelocity = maxV
				}
				if vHData.minVelocity > minV {
					vHData.minVelocity = minV
				}
				newAvg := (avgV + vHData.avgVelocity) / 2
				vHData.avgVelocity = newAvg

				if vHData.topFiveVelocity == nil {
					vHData.topFiveVelocity = make([]float64, 5)
				}

				// Now add top 5 max velocity
				// symbol, maxVelocity, minVelocity, avgVelocity, v1, v2, v3, v4, v5

				if len(record) >= 9 {
					v0, _ := strconv.ParseFloat(record[4], 64)
					v1, _ := strconv.ParseFloat(record[5], 64)
					v2, _ := strconv.ParseFloat(record[6], 64)
					v3, _ := strconv.ParseFloat(record[7], 64)
					v4, _ := strconv.ParseFloat(record[8], 64)

					vHData.topFiveVelocity = updateTopFiveVelocity(v0, vHData.topFiveVelocity)
					vHData.topFiveVelocity = updateTopFiveVelocity(v1, vHData.topFiveVelocity)
					vHData.topFiveVelocity = updateTopFiveVelocity(v2, vHData.topFiveVelocity)
					vHData.topFiveVelocity = updateTopFiveVelocity(v3, vHData.topFiveVelocity)
					vHData.topFiveVelocity = updateTopFiveVelocity(v4, vHData.topFiveVelocity)

				} else if vHData.topFiveVelocity[0] < maxV {
					vHData.topFiveVelocity[0] = maxV
					slices.Sort(vHData.topFiveVelocity)
				}

				avg := (vHData.topFiveVelocity[0] + vHData.topFiveVelocity[1] + vHData.topFiveVelocity[2] + vHData.topFiveVelocity[3] + vHData.topFiveVelocity[4]) / 5.0
				vHData.topFiveVelocityAvg = avg
				m.MarketHistoricalVelocityData[symbol] = vHData

			} else {
				fmt.Printf("this record in file %d is incomplete %v\n", i, record)
			}
		}
		for _, rec := range m.MarketHistoricalVelocityData {
			fmt.Printf("Velocity data symbol from %d:  %s, max: %f, min: %f, avg: %f\n", i, rec.symbol, rec.maxVelocity, rec.minVelocity, rec.avgVelocity)
		}
	}

	fmt.Println("Final historical velocity data to use for the day")
	fmt.Println("&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&")

	for _, vData := range m.MarketHistoricalVelocityData {
		fmt.Println(vData.symbol, vData.maxVelocity, vData.topFiveVelocityAvg, vData.topFiveVelocity)
	}
	fmt.Println("&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&")

}
*/
