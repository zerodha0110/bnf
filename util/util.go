package util

import (
	"fmt"
	"kite/types"
	"log"
	"os"
	"strconv"
)

var envConf *types.EnvConfig

func init() {
	fmt.Println("Init function1111111111111111111111111111111111111111111111111111")
	ReadEnvConfig()
}

func GetEnvConfig() *types.EnvConfig {
	return envConf
}

func ReadEnvConfig() {
	envConf = new(types.EnvConfig) // Add mutex later
	envConf.CFWConf = types.CandleFlowWave{}
	envConf.CHConf = types.CandleHtConf{}
	envConf.GTTConf = types.GttConf{}

	fmt.Println("Start reading env config")
	var err error = nil

	envConf.RequestToken = os.Getenv("REQUEST_TOKEN")
	envConf.Strategy = os.Getenv("STRATEGY")
	envConf.Simulation, err = strconv.ParseBool(os.Getenv("SIMULATION"))
	envConf.TestRun, err = strconv.ParseBool(os.Getenv("TEST_RUN"))
	envConf.TestBnf, err = strconv.ParseFloat(os.Getenv("TEST_BNF"), 64)
	envConf.TestKillJob, err = strconv.ParseBool(os.Getenv("TEST_KILL_JOB"))
	envConf.TestMultiplier, err = strconv.ParseFloat(os.Getenv("TEST_MULTIPLIER"), 64)

	if envConf.Strategy == types.STRATEGY_CANDLE_HEIGHT {
		//Need to implement this...
		envConf.CHConf.LotSize, err = strconv.ParseFloat(os.Getenv("CH_POSITION_LOT_SIZE"), 64)
		envConf.CHConf.Exit_1_Points, err = strconv.ParseFloat(os.Getenv("CH_EXIT_1"), 64)
		envConf.CHConf.Exit_2_Points, err = strconv.ParseFloat(os.Getenv("CH_EXIT_2"), 64)
		envConf.CHConf.IdentifyEntryPrice, err = strconv.ParseBool((os.Getenv("CH_IDENTIFY_ENTRY_PRICE")))
		envConf.CHConf.PositionToManage = os.Getenv("CH_POSITION_TYPE_TO_MANAGE")
		if envConf.CHConf.PositionToManage == "sell" {
			envConf.CHConf.SellPositionExists, err = strconv.ParseBool(os.Getenv("CH_SELL_POSITION_EXISTS"))
			envConf.CHConf.SellPositionEntryPrice, err = strconv.ParseFloat(os.Getenv("CH_SELL_POSITION_ENTRY_PRICE"), 64)
		} else if envConf.CHConf.PositionToManage == "buy" {
			envConf.CHConf.BuyPositionExists, err = strconv.ParseBool(os.Getenv("CH_BUY_POSITION_EXISTS"))
			envConf.CHConf.BuyPositionEntryPrice, err = strconv.ParseFloat(os.Getenv("CH_BUY_POSITION_ENTRY_PRICE"), 64)
		} else {
			fmt.Println("Unknown position to manage", envConf.CHConf.PositionToManage)
			os.Exit(-1)
		}
		envConf.CHConf.StopLossPoints, err = strconv.ParseFloat(os.Getenv("CH_STOP_LOSS_POINTS"), 64)
		if err != nil {
			log.Fatalf("Invalid StopLossPoints: %#v", err)
		}

	} else if envConf.Strategy == types.STRATEGY_CANDLE_FLOW_WAVE {
		envConf.CFWConf.CandleHeight, err = strconv.ParseFloat(os.Getenv("CFW_CANDLE_HEIGHT"), 64)
		if err != nil {
			log.Fatalf("Invalid Candle Height: %v", err)
		}
		envConf.CFWConf.CandleStart, err = strconv.ParseFloat(os.Getenv("CFW_CANDLE_START"), 64)
		if err != nil {
			log.Fatalf("Invalid Candle start: %v", err)
		}
		envConf.CFWConf.CandleColor = os.Getenv("CFW_CANDLE_COLOR")
		if envConf.CFWConf.CandleColor != "green" && envConf.CFWConf.CandleColor != "red" {
			log.Fatalf("Invalid FC Color")
		}
		envConf.CFWConf.LotSize, err = strconv.ParseFloat(os.Getenv("CFW_POSITION_LOT_SIZE"), 64)
		if err != nil {
			log.Fatalf("Invalid positionLotSize: %#v", err)
		}
		envConf.CFWConf.PositionToManage = os.Getenv("CFW_POSITION_TYPE_TO_MANAGE")
		envConf.CFWConf.ProfitPoints, err = strconv.ParseFloat(os.Getenv("CFW_PROFIT_POINTS"), 64)
		if err != nil {
			log.Fatalf("Invalid ProfitPoints: %#v", err)
		}
		envConf.CFWConf.StopLossPoints, err = strconv.ParseFloat(os.Getenv("CFW_STOP_LOSS_POINTS"), 64)
		if err != nil {
			log.Fatalf("Invalid StopLossPoints: %#v", err)
		}
		envConf.CFWConf.ReverseOnStopLoss, err = strconv.ParseBool(os.Getenv("CFW_REVERSE_ON_STOP_LOSS"))
		envConf.CFWConf.TrailingStopLoss, err = strconv.ParseBool(os.Getenv("CFW_TRAILING_STOP_LOSS"))
		envConf.CFWConf.DoubleDown, err = strconv.ParseBool(os.Getenv("CFW_DOUBLE_DOWN"))
		envConf.CFWConf.ExitOnReverse, err = strconv.ParseBool(os.Getenv("CFW_EXIT_ON_REVERSE"))
		envConf.CFWConf.DayHigh, err = strconv.ParseFloat(os.Getenv("CFW_DAY_HIGH"), 64)
		if err != nil {
			log.Fatalf("Invalid Candle day high: %v", err)
		}

		envConf.CFWConf.DayLow, err = strconv.ParseFloat(os.Getenv("CFW_DAY_LOW"), 64)
		if err != nil {
			log.Fatalf("Invalid Candle day low: %v", err)
		}

		envConf.CFWConf.PrevDayClose, err = strconv.ParseFloat(os.Getenv("CFW_PREV_DAY_CLOSE"), 64)
		if err != nil {
			log.Fatalf("Invalid Previous day close: %v", err)
		}
		envConf.CFWConf.TimeBased, err = strconv.ParseBool(os.Getenv("CFW_TIME_BASED"))

	} else if envConf.Strategy == types.STRATEGY_CANDLE_FLOW_WAVE_SHARE {
		envConf.CFWSConf.CandleHeight, err = strconv.ParseFloat(os.Getenv("CFWS_CANDLE_HEIGHT"), 64)
		if err != nil {
			log.Fatalf("Invalid Candle Height: %v", err)
		}
		envConf.CFWSConf.CandleStart, err = strconv.ParseFloat(os.Getenv("CFWS_CANDLE_START"), 64)
		if err != nil {
			log.Fatalf("Invalid Candle start: %v", err)
		}
		envConf.CFWSConf.CandleColor = os.Getenv("CFWS_CANDLE_COLOR")
		if envConf.CFWSConf.CandleColor != "green" && envConf.CFWSConf.CandleColor != "red" {
			log.Fatalf("Invalid FC Color")
		}
		envConf.CFWSConf.LotSize, err = strconv.ParseFloat(os.Getenv("CFWS_POSITION_LOT_SIZE"), 64)
		if err != nil {
			log.Fatalf("Invalid positionLotSize: %#v", err)
		}
		envConf.CFWSConf.PositionToManage = os.Getenv("CFWS_POSITION_TYPE_TO_MANAGE")

		envConf.CFWSConf.DayHigh, err = strconv.ParseFloat(os.Getenv("CFWS_DAY_HIGH"), 64)
		if err != nil {
			log.Fatalf("Invalid Candle day high: %v", err)
		}

		envConf.CFWSConf.DayLow, err = strconv.ParseFloat(os.Getenv("CFWS_DAY_LOW"), 64)
		if err != nil {
			log.Fatalf("Invalid Candle day low: %v", err)
		}

		envConf.CFWSConf.PrevDayClose, err = strconv.ParseFloat(os.Getenv("CFWS_PREV_DAY_CLOSE"), 64)
		if err != nil {
			log.Fatalf("Invalid Previous day close: %v", err)
		}
		envConf.CFWSConf.StopLossPoints, err = strconv.ParseFloat(os.Getenv("CFWS_STOP_LOSS_POINTS"), 64)
		if err != nil {
			log.Fatalf("Invalid StopLossPoints: %#v", err)
		}

		x, err1 := strconv.ParseInt(os.Getenv("CFWS_TICKER_ID"), 10, 32)
		if err1 != nil {
			log.Fatalf("Invalid tickerID")
		}
		envConf.CFWSConf.TickerId = uint32(x)

		envConf.CFWSConf.TickerName = os.Getenv("CFWS_TICKER_NAME")

	} else if envConf.Strategy == types.STRATEGY_CANDLE_HEIGHT_SHARE {

		//Using this as previous day close.. fix it later...
		/*	envConf.CFWSConf.CandleStart, err = strconv.ParseFloat(os.Getenv("CFWS_CANDLE_START"), 64)
			if err != nil {
				log.Fatalf("Invalid Candle start for %s: %v", envConf.Strategy, err)
			}
		*/
		envConf.CFWSConf.LotSize, err = strconv.ParseFloat(os.Getenv("CFWS_POSITION_LOT_SIZE"), 64)
		if err != nil {
			log.Fatalf("Invalid positionLotSize: %#v", err)
		}

		envConf.CFWSConf.PrevDayClose, err = strconv.ParseFloat(os.Getenv("CFWS_PREV_DAY_CLOSE"), 64)
		if err != nil {
			log.Fatalf("Invalid Previous day close: %v", err)
		}
		envConf.CFWSConf.StopLossPoints, err = strconv.ParseFloat(os.Getenv("CFWS_STOP_LOSS_POINTS"), 64)
		if err != nil {
			log.Fatalf("Invalid StopLossPoints: %#v", err)
		}

		x, err1 := strconv.ParseInt(os.Getenv("CFWS_TICKER_ID"), 10, 32)
		if err1 != nil {
			log.Fatalf("Invalid tickerID")
		}
		envConf.CFWSConf.TickerId = uint32(x)

		envConf.CFWSConf.TickerName = os.Getenv("CFWS_TICKER_NAME")

	}
	MyPrintf("End reading env config: %#v", envConf)

}

func MyPrintf(format string, a ...any) {

}
