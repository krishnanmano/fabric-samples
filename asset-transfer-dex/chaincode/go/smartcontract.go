package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

type SmartContract struct {
	contractapi.Contract
}

type Asset struct {
	ID    string  `json:"id"`
	Owner string  `json:"owner"`
	Price float64 `json:"price"`
}

type Order struct {
	ID        string  `json:"id"`
	AssetID   string  `json:"assetId"`
	OrderType string  `json:"orderType"` // "buy" or "sell"
	Price     float64 `json:"price"`
	Owner     string  `json:"owner"`
}

func (s *SmartContract) CreateAsset(ctx contractapi.TransactionContextInterface, id string, owner string, price float64) error {
	asset := Asset{
		ID:    id,
		Owner: owner,
		Price: price,
	}
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(id, assetJSON)
}

func (s *SmartContract) PlaceOrder(ctx contractapi.TransactionContextInterface, id string, assetId string, orderType string, price float64, owner string) error {
	order := Order{
		ID:        id,
		AssetID:   assetId,
		OrderType: orderType,
		Price:     price,
		Owner:     owner,
	}
	orderJSON, err := json.Marshal(order)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(id, orderJSON)
}

func (s *SmartContract) CancelOrder(ctx contractapi.TransactionContextInterface, orderID string) error {
	exists, err := ctx.GetStub().GetState(orderID)
	if err != nil {
		return err
	}
	if exists == nil {
		return fmt.Errorf("order %s does not exist", orderID)
	}
	return ctx.GetStub().DelState(orderID)
}

func (s *SmartContract) MatchAndExecuteTrade(ctx contractapi.TransactionContextInterface, buyOrderID string, sellOrderID string) error {
	buyOrderJSON, err := ctx.GetStub().GetState(buyOrderID)
	if err != nil {
		return err
	}
	sellOrderJSON, err := ctx.GetStub().GetState(sellOrderID)
	if err != nil {
		return err
	}

	if buyOrderJSON == nil || sellOrderJSON == nil {
		return fmt.Errorf("one or both orders do not exist")
	}

	var buyOrder, sellOrder Order
	json.Unmarshal(buyOrderJSON, &buyOrder)
	json.Unmarshal(sellOrderJSON, &sellOrder)

	if buyOrder.Price < sellOrder.Price {
		return fmt.Errorf("orders do not match")
	}

	assetJSON, err := ctx.GetStub().GetState(sellOrder.AssetID)
	if err != nil {
		return err
	}

	var asset Asset
	json.Unmarshal(assetJSON, &asset)

	asset.Owner = buyOrder.Owner
	updatedAssetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	ctx.GetStub().PutState(asset.ID, updatedAssetJSON)
	ctx.GetStub().DelState(buyOrderID)
	ctx.GetStub().DelState(sellOrderID)

	return nil
}

func (s *SmartContract) GetActiveOrders(ctx contractapi.TransactionContextInterface) ([]Order, error) {
	queryString := `{"selector":{"orderType":{"$exists":true}}}`
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var orders []Order
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		var order Order
		if err := json.Unmarshal(queryResponse.Value, &order); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func (s *SmartContract) GetTradeHistory(ctx contractapi.TransactionContextInterface, assetID string) ([]Asset, error) {
	historyIterator, err := ctx.GetStub().GetHistoryForKey(assetID)
	if err != nil {
		return nil, err
	}
	defer historyIterator.Close()

	var history []Asset
	for historyIterator.HasNext() {
		historyResponse, err := historyIterator.Next()
		if err != nil {
			return nil, err
		}
		var asset Asset
		if err := json.Unmarshal(historyResponse.Value, &asset); err == nil {
			history = append(history, asset)
		}
	}
	return history, nil
}

func (s *SmartContract) GetAsset(ctx contractapi.TransactionContextInterface, id string) (*Asset, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, err
	}
	if assetJSON == nil {
		return nil, fmt.Errorf("asset %s does not exist", id)
	}
	var asset Asset
	err = json.Unmarshal(assetJSON, &asset)
	if err != nil {
		return nil, err
	}
	return &asset, nil
}
