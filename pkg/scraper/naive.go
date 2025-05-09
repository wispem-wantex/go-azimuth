package scraper

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	. "go-azimuth/pkg/db"
)

// Fetch all the Naive logs, and then fetch the transaction data for each log
func CatchUpNaiveLogs(client *ethclient.Client, db DB) {
	latest_block, err := client.BlockNumber(context.Background())
	if err != nil {
		panic(err)
	}

	contract := db.GetContractByName("Naive")

	// Assume we can fetch all the logs in 1 query
	// TODO: not a good assumption
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(0).SetUint64(max(contract.StartBlockNum, contract.LatestBlockNumFetched)),
		ToBlock:   big.NewInt(0).SetUint64(latest_block),
		Addresses: []common.Address{contract.Address},
	}
	logs, err := client.FilterLogs(context.Background(), query)
	if err != nil {
		// TODO: this would be where to handle if there's too many logs for 1 query
		panic(err)
	}

	// To get Tx data, we have to use batching; otherwise, turbo slow
	parsed_logs := []EthereumEventLog{}

	// First, save the Ethereum logs
	for _, l := range logs {
		fmt.Println("----------")
		fmt.Println("Log Block Number:", l.BlockNumber)
		fmt.Printf(EVENT_NAMES[l.Topics[0]])
		fmt.Printf("(")
		for _, t := range l.Topics[1:] {
			fmt.Print(t, ", ")
		}
		fmt.Printf(")\n")
		if len(l.Data) != 0 {
			fmt.Println(hex.EncodeToString(l.Data))
		}

		naive_event_log := ParseEthereumLog(l)
		// Save it in the DB
		db.SaveEvent(&naive_event_log)

		// Add it to the list of call-data to fetch
		parsed_logs = append(parsed_logs, naive_event_log)
	}

	GetNaiveTransactionData(client, db, parsed_logs)
	db.SetLatestContractBlockFetched(contract.ID, latest_block)
}

// Get transaction data (call-data) for Batch events, in batches (yes)
func GetNaiveTransactionData(client *ethclient.Client, db DB, logs []EthereumEventLog) {
	contract := db.GetContractByName("Naive")
	// Callback function to execute RPC batches
	do_batched_rpc := func(batch []rpc.BatchElem) []*types.Transaction {
		if err := client.Client().BatchCall(batch); err != nil {
			log.Fatalf("Batch call failed: %v", err)
		}

		ret := []*types.Transaction{} // Has to be pointer type to avoid copying an atomic.Pointer
		for _, elem := range batch {
			for _, is_err_service_temp_unavailable := check_error(elem.Error); is_err_service_temp_unavailable; {
				fmt.Printf("Service temporarily unavailable error.  Pausing 1s and trying again\n")
				time.Sleep(1 * time.Second)
				// Try again on temporarily-unavailable errors
				if err := client.Client().BatchCall([]rpc.BatchElem{elem}); err != nil {
					log.Fatalf("Batch call failed: %#v", err)
				}
			}
			// rpc.BatchElem{
			// 	Method:"eth_getTransactionByHash",
			// 	Args:[]interface {}{0xad2f676e4c35c7271123e77bd5616d4e89ae0e93cd0f5a9e4fb93a735ded42be},
			// 	Result:(*types.Transaction)(0xc000332180),
			// 	Error: &rpc.jsonError{Code:-32603, Message:"service temporarily unavailable", Data:interface {}(nil)},
			// }
			if elem.Error != nil {
				panic(fmt.Sprintf("%#v (%#v)", elem, elem.Error))
			}

			tx, is_ok := elem.Result.(*types.Transaction)
			if !is_ok {
				panic(elem.Result)
			}
			ret = append(ret, tx)
		}
		return ret
	}

	// Construct batches
	const MAX_BATCH_SIZE = 20
	for i := 0; i < len(logs); i += MAX_BATCH_SIZE {
		// Compute batch set upper-bound
		ii := i + MAX_BATCH_SIZE
		if ii > len(logs) {
			ii = len(logs)
		}

		fmt.Printf("===================================\ni: %d\n======================================\n\n", i)
		fmt.Printf("Naive contract: fetching tx data %d - %d; total txs is %d\n", i, ii, len(logs))

		// Prepare a batch for RPC-ing
		batch := []rpc.BatchElem{}
		// Index to let us map back from tx hash to update the logs
		logs_by_txhash := make(map[common.Hash]EthereumEventLog)
		for _, l := range logs[i:ii] {
			batch = append(batch,
				rpc.BatchElem{
					Method: "eth_getTransactionByHash",
					Args:   []interface{}{l.TxHash},
					Result: new(types.Transaction), // Use go-ethereum's Transaction struct
				},
			)

			// Also build the index of tx hashes to logs, while we're here
			logs_by_txhash[l.TxHash] = l
		}

		// Execute the batch
		txs := do_batched_rpc(batch)

		// Save the result in the DB
		for _, tx := range txs {
			log, is_ok := logs_by_txhash[tx.Hash()]
			if !is_ok {
				panic(tx.Hash())
			}
			log.Data = tx.Data()
			db.SmuggleNaiveBatchDataIntoEvent(log)
		}

		// Update latest-block-fetched
		// WTF: transactions do not expose a block number, and you can't get it with ethclient
		// See: https://github.com/ethereum/go-ethereum/issues/15210
		db.SetLatestContractBlockFetched(contract.ID, logs[ii-1].BlockNumber)

		time.Sleep(1 * time.Second)
	}
}
