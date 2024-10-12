package scraper

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	. "go-azimuth/pkg/db"
)

const (
	AZIMUTH_ADDRESS            = "0x223c067f8cf28ae173ee5cafea60ca44c335fecb"
	AZIMUTH_START_BLOCK_NUMBER = 6784880
)

// This is insane, but it's the recommended way to do it.
func check_error(err error) (int64, bool) {
	var rpc_err interface {
		Error() string
		ErrorCode() int
		ErrorData() interface{}
	}
	if errors.As(err, &rpc_err) && rpc_err.ErrorCode() == -32005 {
		if rpc_err.ErrorCode() == -32005 {
			data, is_ok := rpc_err.ErrorData().(map[string]interface{})
			if !is_ok {
				panic(data)
			}
			_to_block_recommend, is_ok := data["to"]
			if !is_ok {
				panic(data)
			}
			to_block_recommend, is_ok := _to_block_recommend.(string)
			if !is_ok {
				panic(_to_block_recommend)
			}
			ret, err := strconv.ParseInt(to_block_recommend[2:], 16, 64)
			if err != nil {
				panic(err)
			}
			return ret, true
		}
	}
	return 0, false
}

// Convert it to Our Type, with sanity checks
func ParseEthereumLog(l types.Log) EthereumEventLog {
	event := EthereumEventLog{
		BlockNumber: l.BlockNumber,
		BlockHash:   l.BlockHash,
		TxHash:      l.TxHash,
		LogIndex:    l.Index,

		ContractAddress: l.Address,
		Name:            EVENT_NAMES[l.Topics[0]],
		Data:            l.Data,
	}

	event.Topic0 = l.Topics[0] // Must exist
	if len(l.Topics) > 1 {
		event.Topic1 = l.Topics[1]
	}
	if len(l.Topics) > 2 {
		event.Topic2 = l.Topics[2]
	}

	return event
}

// Fetches all Azimuth logs since the contract was deployed, in chunks.
func CatchUpAzimuthLogs(client *ethclient.Client, db DB, apply_logs bool) {
	latest_block, err := client.BlockNumber(context.Background())
	if err != nil {
		panic(err)
	}

	batch_size := int64(100000)
	from_block := big.NewInt(AZIMUTH_START_BLOCK_NUMBER)
	to_block := big.NewInt(0).Add(from_block, big.NewInt(batch_size-1))
	contractAddress := common.HexToAddress(AZIMUTH_ADDRESS)
	for i := 0; i < 1000 && from_block.Uint64() < latest_block; i++ {
		query := ethereum.FilterQuery{
			FromBlock: from_block,
			ToBlock:   to_block,
			Addresses: []common.Address{contractAddress},
		}
		logs, err := client.FilterLogs(context.Background(), query)
		if err != nil {
			if to_block_recommend, is_ok := check_error(err); is_ok {
				to_block.SetInt64(to_block_recommend)
				batch_size = to_block_recommend - from_block.Int64()
				continue
			}
			log.Fatalf("Failed to fetch logs: %#v", err)
		}
		// Process the logs
		for _, l := range logs {
			// fmt.Println("----------")
			// fmt.Println("Log Block Number:", l.BlockNumber)
			// fmt.Printf(EVENT_NAMES[l.Topics[0]])
			// fmt.Printf("(")
			// for _, t := range l.Topics[1:] {
			// 	fmt.Print(t, ", ")
			// }
			// fmt.Printf(")\n")
			// if len(l.Data) != 0 {
			// 	fmt.Println(hex.EncodeToString(l.Data))
			// }

			azimuth_event_log := ParseEthereumLog(l)
			if azimuth_event_log.Name == "" {
				// Probably an Ecliptic log
				continue
			}
			db.SaveEvent(&azimuth_event_log)
			if apply_logs {
				db.ApplyEventEffects([]EthereumEventLog{azimuth_event_log})
			}
		}

		// Compute next batch size adaptively
		if len(logs) < 1000 {
			batch_size *= 2
		}

		fmt.Printf("old from: %d; old to: %d\n", from_block, to_block)

		from_block.Add(to_block, big.NewInt(1))
		to_block.Add(from_block, big.NewInt(batch_size-1))
		fmt.Printf("new from: %d; new to: %d\n", from_block, to_block)
		fmt.Printf("===================================\ni: %d\n======================================\n\n", i)
		time.Sleep(1 * time.Second)
	}
}
