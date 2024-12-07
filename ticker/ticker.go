package myTicker

import (
	"fmt"
	"kite/types"
	"math"
	"math/rand"
	"time"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
	kitemodels "github.com/zerodha/gokiteconnect/v4/models"
	kiteticker "github.com/zerodha/gokiteconnect/v4/ticker"
)

type myBNF struct {
	CurrentTick kitemodels.Tick
	updated     bool // there should be better to check if CurrentTick is nil or not.. TODO...
	//We can get these two from Ticker only.. If we use quote or full mode instead of LTP mode - refer onConnect func
	dayHigh         float64
	dayLow          float64
	currentVelocity int
	TickerId        uint32
}

type myFinNifty struct {
	CurrentTick kitemodels.Tick
	updated     bool // there should be better to check if CurrentTick is nil or not.. TODO...
	//We can get these two from Ticker only.. If we use quote or full mode instead of LTP mode - refer onConnect func
	dayHigh         float64
	dayLow          float64
	currentVelocity int
	TickerId        uint32
}

var finNiftyTickerIds []uint32
var finNifty bool

var (
	myTicker         *kiteticker.Ticker
	myBNFInstance    *myBNF
	myFinNFInstances []*myFinNifty
	tickCount        int = 0
)

func updateBNFDetails(tick kitemodels.Tick) {
	myBNFInstance.CurrentTick = tick
	//myBNFInstance.updated = true // Use mutex later TODO. Also initialize myBNFInstance here and remove updated variable
	/*	if tick.LastPrice > myBNFInstance.dayHigh {
			//fmt.Println("It's day high", tick.LastPrice)
			myBNFInstance.dayHigh = tick.LastPrice
		} else if tick.LastPrice < myBNFInstance.dayLow {
			//fmt.Println("It's day low", tick.LastPrice)
			myBNFInstance.dayLow = tick.LastPrice
		}
	*/
}

func updateFinNiftyDetails(tick kitemodels.Tick) {
	instrumentToken := tick.InstrumentToken
	//fmt.Println("intoken", instrumentToken)
	for _, finInstance := range myFinNFInstances {
		if finInstance.TickerId == instrumentToken {
			finInstance.CurrentTick = tick
		}
	}
}

func GetFinNiftyCurrentTick(tickerId uint32) float64 {
	for _, finInstance := range myFinNFInstances {
		if finInstance.TickerId == tickerId {
			return finInstance.CurrentTick.LastPrice
		}
	}
	return 0.0
}

func GetFinNiftyAveragePriceTick(tickerId uint32) float64 {
	for _, finInstance := range myFinNFInstances {
		if finInstance.TickerId == tickerId {
			return finInstance.CurrentTick.AverageTradePrice
		}
	}
	return 0.0
}

func GetFinNiftyCurrentVolume(tickerId uint32) (float64, time.Time) {
	for _, finInstance := range myFinNFInstances {
		if finInstance.TickerId == tickerId {
			return float64(finInstance.CurrentTick.VolumeTraded), finInstance.CurrentTick.LastTradeTime.Time
		}
	}
	return 0.0, time.Time{}
}

func init() {
	myBNFInstance = new(myBNF)
	myBNFInstance.updated = false
	myFinNFInstances = make([]*myFinNifty, 0)

	fmt.Println("!!!!!!!!! myfinnstinstances init")
	finNifty = true
}

func SetTickerIDArray(ids []uint32) {
	finNiftyTickerIds = ids
	for _, tickerId := range finNiftyTickerIds {
		temp := new(myFinNifty)
		temp.TickerId = tickerId
		myFinNFInstances = append(myFinNFInstances, temp)

	}
}

func SetFinNifty(flag bool) {
	finNifty = flag
}

func SetTickerID(id uint32) {
	myBNFInstance.TickerId = id
}

func GetTickerID() uint32 {
	if myBNFInstance.TickerId != 0 {
		return myBNFInstance.TickerId
	}
	return 0
}

func SetDayHighLow(high, low float64) {
	myBNFInstance.dayHigh = high
	myBNFInstance.dayLow = low
}

func GetDayHigh() float64 {
	return myBNFInstance.dayHigh
}

func GetDayLow() float64 {
	return myBNFInstance.dayLow
}

func GetcurrentVelocity() int {
	return myBNFInstance.currentVelocity
}

func GetBNFDetails() *myBNF {
	/*	envConf := util.GetEnvConfig()
		if envConf.TestRun {
			//fmt.Println("Call test")
			return GetBNFDetailsTest(envConf)
		}
		for !myBNFInstance.updated {
			time.Sleep(2 * time.Second)
		}
	*/
	return myBNFInstance
}

func GetCurrentTick() float64 {
	return GetBNFDetails().CurrentTick.LastPrice
}

func GetPreviousClose() float64 {
	return GetBNFDetails().CurrentTick.OHLC.Close
}

func GetPreviousCloseFinNifty(tickerId uint32) float64 {
	for _, finInstance := range myFinNFInstances {
		if finInstance.TickerId == tickerId {
			return finInstance.CurrentTick.OHLC.Close
		}
	}
	return 0.0
}

func GetBNFDetailsTest(envConf *types.EnvConfig) *myBNF {
	rand.Seed(time.Now().UnixNano())
	x := rand.Intn(10)
	y := rand.Intn(4)
	multiplier := -1.0
	if envConf.TestMultiplier == 0.0 {
		if y%2 == 0 {
			multiplier = 1.0
		}
	} else {
		multiplier = envConf.TestMultiplier
	}
	myBNFInstance.CurrentTick.LastPrice = envConf.TestBnf + (float64(x) * multiplier)
	envConf.TestBnf = myBNFInstance.CurrentTick.LastPrice
	fmt.Println(envConf.TestBnf)
	return myBNFInstance
}

// Triggered when any error is raised
func onError(err error) {
	fmt.Println("Error: ", err)
}

// Triggered when websocket connection is closed
func onClose(code int, reason string) {
	fmt.Println("Close: ", code, reason)
}

// Triggered when connection is established and ready to send and accept data
func onConnect() {
	fmt.Println("Connected at ", time.Now().Hour(), time.Now().Minute())

	//err := myTicker.Subscribe([]uint32{GetTickerID()})
	fmt.Println("subscribe subscribe ", finNiftyTickerIds)
	err := myTicker.Subscribe(finNiftyTickerIds)
	if err != nil {
		fmt.Println("err: ", err)
	}
	// Set subscription mode for given list of tokens
	// Default mode is Quote

	if finNifty {
		fmt.Println("connect connect ", finNiftyTickerIds)
		err = myTicker.SetMode(kiteticker.ModeFull, finNiftyTickerIds)
	} else {
		err = myTicker.SetMode(kiteticker.ModeFull, []uint32{GetTickerID()})
	}
	if err != nil {
		fmt.Println("err: ", err)
	}
}

// Triggered when tick is recevived
func onTick(tick kitemodels.Tick) {

	if !finNifty {
		updateBNFDetails(tick)
	} else {
		updateFinNiftyDetails(tick)
		//if tick.InstrumentToken == 98049 {
		//	fmt.Printf("Printing for BEL %v\n", tick.LastTradeTime.Time, tick.LastPrice, tick.VolumeTraded)
		//}
	}

	//tickCount = tickCount + 1
	//if tickCount%2 != 0 {
	//	fmt.Println("Ignore...", tick.LastPrice)
	//return
	//	} else {
	//fmt.Println("Consider...", tick.LastPrice)
	//	tickCount = 0
	//}
	//	fmt.Println("Tick: ", tick)
}

// Triggered when reconnection is attempted which is enabled by default
func onReconnect(attempt int, delay time.Duration) {
	fmt.Printf("Reconnect attempt %d in %fs\n", attempt, delay.Seconds())
}

// Triggered when maximum number of reconnect attempt is made and the program is terminated
func onNoReconnect(attempt int) {
	fmt.Printf("Maximum no of reconnect attempt reached: %d", attempt)
}

// Triggered when order update is received
func onOrderUpdate(order kiteconnect.Order) {
	//fmt.Printf("Order: ", order.OrderID)
}

func StartTicker(ticker *kiteticker.Ticker) {

	// Create new Kite myTicker instance
	myTicker = ticker

	// Assign callbacks
	myTicker.OnError(onError)
	myTicker.OnClose(onClose)
	myTicker.OnConnect(onConnect)
	myTicker.OnReconnect(onReconnect)
	myTicker.OnNoReconnect(onNoReconnect)
	myTicker.OnTick(onTick)
	myTicker.OnOrderUpdate(onOrderUpdate)

	// Start the connection
	myTicker.Serve()
	myBNFInstance.updated = true
	//	go startVelocityMeasurement()
}

func startVelocityMeasurement() {
	time.Sleep(10 * time.Second)
	for {
		tick1 := myBNFInstance.CurrentTick.LastPrice
		time.Sleep(60 * time.Second)
		tick2 := myBNFInstance.CurrentTick.LastPrice
		if math.Abs(tick1-tick2) > 25 {
			if tick1 > tick2 {
				//It's falling
				myBNFInstance.currentVelocity = -1
				fmt.Println("Time", time.Now().Hour(), time.Now().Minute())
				fmt.Printf("Went down from %f to %f", tick1, tick2)
			} else {
				fmt.Println("Time", time.Now().Hour(), time.Now().Minute())
				fmt.Printf("Went up from %f to %f", tick1, tick2)
				myBNFInstance.currentVelocity = 1
			}
		} else {
			myBNFInstance.currentVelocity = 0
		}
	}
}
