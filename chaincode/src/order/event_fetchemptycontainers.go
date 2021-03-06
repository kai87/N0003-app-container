// Copyright [2016] [Cuiting Shi ]
// 
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// 
// http://www.apache.org/licenses/LICENSE-2.0
// 
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// 
package order

import (
	"fmt"
	"encoding/json"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// OrderFetchEmptyContainersRequest - fetch empty containers request
type OrderFetchEmptyContainersRequest struct {
	CarrierId        string `json:"carrierId"`
	OrderId          string `json:"orderid"`
	DateForDelivered string `json:"datefordelivered"` // 设备移出时间
}

// =============================================================================
// FetchEmptyContainers - fetch empty containers
// arguments:
// 	0,
// 	OrderFetchEmptyContainersRequest,
//
// response:
//	0,
//	orderId
// =============================================================================

func FetchEmptyContainer(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	logger.Info("Starting fetchEmptyContainer")
	logger.Infof("Receive %d arguments for fetchEmptyContainer: %v\n", len(args), args)

	var err error
	fmt.Println("starting checkOrder")

	err = CheckArguments(args, 1)
	if err != nil {
		return nil, err
	}
	request := new(OrderFetchEmptyContainersRequest)
	err = json.Unmarshal([]byte(args[0]), request)
	if err != nil {
		logger.Error(NewOrderErrorMessage(ERROR_ARGUMENTS, "Incorrect type, expecting %s", request.GetType()))
		return nil, NewOrderErrorMessage(ERROR_ARGUMENTS, "Incorrect type, expecting %s", request.GetType())
	}
	logger.Infof("Fetch empty containers, unmarshal request: %+v",request)


	order := new(Order)
	if isValid, err := request.isValid(stub, order); !isValid {
		return nil, err
	}

	// change the state

	handleFSM := NewOrderHandleFSM(stub, request.CarrierId, request.OrderId, order.State)

	err = handleFSM.FSM.Event(EVENT_FETCH_EMPTY_CONTAINERS)
	if err != nil {
		return nil, NewShimError(ERROR_FSM, "Failed to checkOrder: %v", err)
	}
	order.State = handleFSM.FSM.Current()

	// update carrying form which the carrier is in charge of
	order.CarryingForm.DateForDeliver = request.DateForDelivered
	order.CarryingForm.Status = handleFSM.FSM.Current()

	// write order back into the ledger
	if err = order.PutOrder(stub); err != nil {
		return nil, NewShimError(ERROR_INTERNAL, err.Error())
	}
	logger.Infof("Fetched empty containers, order is %v", order)

	//create message to notify the client
	err = SendMessage(stub, order.ConsigningForm.ClientId, "Order %s has been processed by carrier, with empty container fetched. Further details are displayed in the order platform.", order.OrderNo)
	if err != nil {
		logger.Warningf("Failed to send message to user %s: %v", order.ConsigningForm.CargoAgentId, err)
	}

	fmt.Println("- end fetchEmptyContainers")
	return []byte(order.Id), nil
}

func (request *OrderFetchEmptyContainersRequest) isValid(stub shim.ChaincodeStubInterface, order *Order) (bool, error) {
	if !IsOrderExist(stub, request.OrderId, order) {
		return false, NewOrderErrorMessage(ERROR_REQUEST, "Order %s not exist", request.OrderId)
	}

	if order.CarryingForm.CarrierId != request.CarrierId {
		return false, NewOrderErrorMessage(ERROR_REQUEST, "Carrier %s can't modify order %v", request.CarrierId, order)
	}
	return true, nil
}

func (request *OrderFetchEmptyContainersRequest) GetType() string {
	return fmt.Sprintf("%T", *request)
}
