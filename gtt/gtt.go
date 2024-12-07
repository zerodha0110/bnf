package gtt

import (
	"kite/types"
	"log"
	"time"

	myTicker "kite/ticker"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

func GetGTT(kc *kiteconnect.Client, gttId int) (kiteconnect.GTT, error) {
	retryCount := types.GTTRetryCount
	var err error = nil
	var gttResp kiteconnect.GTT

	for retryCount != 0 {
		gttResp, err = kc.GetGTT(gttId)
		if err != nil {
			log.Printf("error getting  gtt: %v", err)
			log.Printf("Time of error is:%d - %d", time.Now().Hour(), time.Now().Minute())
			retryCount = retryCount - 1
			time.Sleep(time.Second * 2)
		} else {
			//log.Printf("gtt status is: %v", gttResp.Status)
			break
		}
	}
	return gttResp, err
}

func DeleteGTT(kc *kiteconnect.Client, gttId int) (kiteconnect.GTTResponse, error) {
	retryCount := types.GTTRetryCount
	var err error = nil
	var gttResp kiteconnect.GTTResponse

	for retryCount != 0 {
		gttResp, err = kc.DeleteGTT(gttId)
		if err != nil {
			log.Printf("error getting  gtt: %v", err)
			retryCount = retryCount - 1
			time.Sleep(time.Second * 2)
		} else {
			//log.Printf("gtt status is: %v", gttResp.Status)
			break
		}
	}
	return gttResp, err
}

func ModifyGTT(kc *kiteconnect.Client, gttId int, transactionType string, target, lotSize, limitPrice float64) (kiteconnect.GTTResponse, error) {

	retryCount := types.GTTRetryCount
	var err error = nil
	var gttModifyResp kiteconnect.GTTResponse

	for retryCount != 0 {
		bnfDetails := myTicker.GetBNFDetails()
		currentTick := bnfDetails.CurrentTick.LastPrice
		log.Printf("Modify GTT for target:%f , currentTick %f, retryCount:%d, triggerID:%d", target, currentTick, retryCount, gttId)
		limitPrice = float64(int(limitPrice))
		target := float64(int(target))
		gttModifyResp, err = kc.ModifyGTT(gttId, kiteconnect.GTTParams{
			Tradingsymbol:   types.BankNiftySymbol,
			Exchange:        "NFO",
			LastPrice:       currentTick,
			TransactionType: transactionType,
			Trigger: &kiteconnect.GTTSingleLegTrigger{
				TriggerParams: kiteconnect.TriggerParams{
					TriggerValue: target,
					Quantity:     lotSize,
					LimitPrice:   limitPrice,
				},
			},
		})
		if err != nil {
			log.Printf("error modifying  gtt: %v", err)
			retryCount = retryCount - 1
			time.Sleep(time.Second * 2)
		} else {
			log.Println("Modified GTT trigger_id = ", gttModifyResp.TriggerID)
			break
		}
	}
	return gttModifyResp, err
}

func PlaceGTTWithBuffer(kc *kiteconnect.Client, transactionType string, target, lotSize, limitPrice float64) (kiteconnect.GTTResponse, error) {
	if transactionType == kiteconnect.TransactionTypeBuy {
		limitPrice = limitPrice + types.TransactionBuffer // Buy for little more
	} else if transactionType == kiteconnect.TransactionTypeSell {
		limitPrice = limitPrice - types.TransactionBuffer // Sell for little less
	}
	return PlaceGTT(kc, transactionType, target, lotSize, limitPrice)
}

func ModifyGTTWithBuffer(kc *kiteconnect.Client, gttId int, transactionType string, target, lotSize, limitPrice float64) (kiteconnect.GTTResponse, error) {
	if transactionType == kiteconnect.TransactionTypeBuy {
		limitPrice = limitPrice + types.TransactionBuffer // Buy for little more
	} else if transactionType == kiteconnect.TransactionTypeSell {
		limitPrice = limitPrice - types.TransactionBuffer // Sell for little less
	}
	return ModifyGTT(kc, gttId, transactionType, target, lotSize, limitPrice)
}

func PlaceGTT(kc *kiteconnect.Client, transactionType string, target, lotSize, limitPrice float64) (kiteconnect.GTTResponse, error) {

	retryCount := types.GTTRetryCount
	var err error = nil
	var gttResp kiteconnect.GTTResponse

	for retryCount != 0 {
		bnfDetails := myTicker.GetBNFDetails()
		currentTick := bnfDetails.CurrentTick.LastPrice
		log.Printf("Place new GTT for target:%f , currentTick %f, retryCount:%d", target, currentTick, retryCount)
		limitPrice = float64(int(limitPrice))
		target := float64(int(target))
		gttResp, err = kc.PlaceGTT(kiteconnect.GTTParams{
			Tradingsymbol:   types.BankNiftySymbol,
			Exchange:        "NFO",
			LastPrice:       currentTick,
			TransactionType: transactionType,
			Trigger: &kiteconnect.GTTSingleLegTrigger{
				TriggerParams: kiteconnect.TriggerParams{
					TriggerValue: target,
					Quantity:     lotSize,
					LimitPrice:   limitPrice,
				},
			},
		})
		if err != nil {
			log.Printf("error placing  gtt: %v", err)
			retryCount = retryCount - 1
			time.Sleep(time.Second * 2)
		} else {
			log.Println("placed GTT trigger_id = ", gttResp.TriggerID)
			break
		}
	}
	return gttResp, err
}
