package types

import (
	"fmt"
	"time"
)

//https://kite.trade/connect/login?api_key=pshndp9jhkxo36ic&v=3
//https://developers.kite.trade/apps/pshndp9jhkxo36ic
//https://api.kite.trade/instruments

/*
https://api.kite.trade/instruments
*/

const (
	BankNiftySymbol                 string        = "BANKNIFTY24JULFUT"
	SLStartPercentage               float64       = 0.0040
	DeadEndStopLossPercentage       float64       = 0.0040
	DeadEndStopLossPoints           float64       = 300.00 // Roughly 0.4% of 38000
	GTTRetryCount                   int           = 20
	GTTTriggeredStatus              string        = "triggered"
	GTTDeletedStatus                string        = "deleted"
	CheckInterval                   time.Duration = 30
	STRATEGY_GTT                                  = "GTT"
	STRATEGY_TWO_LEG_EXIT                         = "TWO_LEG_EXIT"
	STRATEGY_CANDLE_HEIGHT                        = "CANDLE_HEIGHT"
	STRATEGY_CANDLE_FLOW_WAVE                     = "CANDLE_FLOW_WAVE"
	STRATEGY_CANDLE_FLOW_WAVE_SHARE               = "CANDLE_FLOW_WAVE_SHARE"
	STRATEGY_CANDLE_HEIGHT_SHARE                  = "CANDLE_HEIGHT_SHARE"
	STRATEGY_TICK_FLOW_WAVE                       = "TICK_FLOW_WAVE"
	TransactionBuffer               float64       = 1.0
	TwentyPointBuffer               float64       = 30.0
	BorkeragePoints                 float64       = -2.0
)

// Move GTT/Order related fields to individual struts later TODO
type EnvConfig struct {
	RequestToken   string
	TestRun        bool
	TestBnf        float64
	TestKillJob    bool
	TestMultiplier float64
	Strategy       string
	CHConf         CandleHtConf
	CFWConf        CandleFlowWave
	CFWSConf       CandleFlowWaveShare
	GTTConf        GttConf
	TFWConf        TickFlowWave
	Simulation     bool
}

type CandleHtConf struct {
	BuyPositionExists      bool
	BuyPositionEntryPrice  float64
	SellPositionExists     bool
	SellPositionEntryPrice float64
	LotSize                float64
	Exit_1_Points          float64
	Exit_2_Points          float64
	IdentifyEntryPrice     bool
	PositionToManage       string
	StopLossPoints         float64
}

type GttConf struct {
	PositionExists bool

	TransactionType    string
	PositionEntryPrice float64
	LotSize            float64

	PlaceOrder               bool
	PlaceSellOrderEntryPrice float64
	PlaceBuyOrderEntryPrice  float64
	PlaceSellOrderQty        float64
	PlaceBuyOrderQty         float64
}

type CandleFlowWave struct {
	CandleHeight      float64
	CandleStart       float64
	CandleColor       string
	LotSize           float64
	PositionToManage  string
	ProfitPoints      float64
	StopLossPoints    float64
	ReverseOnStopLoss bool
	TrailingStopLoss  bool
	DoubleDown        bool
	DayHigh           float64
	DayLow            float64
	PrevDayClose      float64
	ExitOnReverse     bool
	TimeBased         bool
}

type CandleFlowWaveShare struct {
	CandleHeight     float64
	CandleStart      float64
	CandleColor      string
	LotSize          float64
	PositionToManage string
	StopLossPoints   float64
	DayHigh          float64
	DayLow           float64
	PrevDayClose     float64
	TickerId         uint32
	TickerName       string
	BankNiftyLotSize float64
	ReservedMargin   float64
}

type TickFlowWave struct {
	CandleStart      float64
	CandleColor      string
	LotSize          float64
	PositionToManage string
	ExecutionType    string
	Algo             string
}

type CandleDetails struct {
	Height    float64
	StartTick float64
	EndTick   float64
	Color     string
}

func (cd CandleDetails) String() string {
	return fmt.Sprintf("Color:%s, StartTick:%f, EndTick:%f, Height:%f", cd.Color, cd.StartTick, cd.EndTick, cd.Height)
}

// TODO - Just keep profit, ronds, profitRounds, lossRounds.. and create 2 entities for BuyMetrics and SellMetrics.
type Metrics struct {
	BuyProfit         float64
	SellProfit        float64
	BuyRounds         int
	SellRounds        int
	BuyProfitRounds   int
	BuyLossRounds     int
	SellProfitRounds  int
	SellLossRounds    int
	TotalProfitPoints float64
}

func (cd Metrics) String() string {
	return fmt.Sprintf("Metrics :%#v", cd)
}
