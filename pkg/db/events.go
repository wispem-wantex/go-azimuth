package db

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/crypto/sha3"
)

const (
	AZIMUTH_ADDRESS    = "0x223c067f8cf28ae173ee5cafea60ca44c335fecb"
	START_BLOCK_NUMBER = 6784880
)

func get_hash(s string) common.Hash {
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(s))
	return common.Hash(hash.Sum(nil))
}

var (
	// Azimuth Events
	SPAWNED                  = get_hash("Spawned(uint32,uint32)")
	ACTIVATED                = get_hash("Activated(uint32)")
	OWNER_CHANGED            = get_hash("OwnerChanged(uint32,address)")
	CHANGED_SPAWN_PROXY      = get_hash("ChangedSpawnProxy(uint32,address)")
	CHANGED_TRANSFER_PROXY   = get_hash("ChangedTransferProxy(uint32,address)")
	CHANGED_MANAGEMENT_PROXY = get_hash("ChangedManagementProxy(uint32,address)")
	CHANGED_VOTING_PROXY     = get_hash("ChangedVotingProxy(uint32,address)")
	ESCAPE_REQUESTED         = get_hash("EscapeRequested(uint32,uint32)")
	ESCAPE_CANCELED          = get_hash("EscapeCanceled(uint32,uint32)")
	ESCAPE_ACCEPTED          = get_hash("EscapeAccepted(uint32,uint32)")
	LOST_SPONSOR             = get_hash("LostSponsor(uint32,uint32)")
	BROKE_CONTINUITY         = get_hash("BrokeContinuity(uint32,uint32)")
	CHANGED_KEYS             = get_hash("ChangedKeys(uint32,bytes32,bytes32,uint32,uint32)")
	CHANGED_DNS              = get_hash("ChangedDns(string,string,string)")

	// Ecliptic Events
	// APPROVAL_FOR_ALL         = get_hash("ApprovalForAll(address,address,bool)")
	// OWNERSHIP_TRANSFERRED    = get_hash("OwnershipTransferred(address,address)")

	// Naive Events (TODO)
)

var EVENT_NAMES = map[common.Hash]string{}

func init() {
	EVENT_NAMES[OWNER_CHANGED] = "OwnerChanged"
	EVENT_NAMES[ACTIVATED] = "Activated"
	EVENT_NAMES[SPAWNED] = "Spawned"
	EVENT_NAMES[ESCAPE_REQUESTED] = "EscapeRequested"
	EVENT_NAMES[ESCAPE_CANCELED] = "EscapeCanceled"
	EVENT_NAMES[ESCAPE_ACCEPTED] = "EscapeAccepted"
	EVENT_NAMES[LOST_SPONSOR] = "LostSponsor"
	EVENT_NAMES[CHANGED_KEYS] = "ChangedKeys"
	EVENT_NAMES[BROKE_CONTINUITY] = "BrokeContinuity"
	EVENT_NAMES[CHANGED_SPAWN_PROXY] = "ChangedSpawnProxy"
	EVENT_NAMES[CHANGED_TRANSFER_PROXY] = "ChangedTransferProxy"
	EVENT_NAMES[CHANGED_MANAGEMENT_PROXY] = "ChangedManagementProxy"
	EVENT_NAMES[CHANGED_VOTING_PROXY] = "ChangedVotingProxy"
	EVENT_NAMES[CHANGED_DNS] = "ChangedDns"

	// EVENT_NAMES[APPROVAL_FOR_ALL] = "ApprovalForAll"
	// EVENT_NAMES[OWNERSHIP_TRANSFERRED] = "OwnershipTransferred"
}

type Query struct {
	SQL        string
	BindValues interface{}
}

type AzimuthEventLog struct {
	BlockNumber uint64      `db:"block_number"`
	BlockHash   common.Hash `db:"block_hash"`
	TxHash      common.Hash `db:"tx_hash"`
	LogIndex    uint        `db:"log_index"`

	ContractAddress common.Address `db:"contract_address"`
	Name            string         `db:"name"`
	Topic0          common.Hash    `db:"topic0"` // Hashed version of Name and the arg types
	Topic1          common.Hash    `db:"topic1"`
	Topic2          common.Hash    `db:"topic2"`
	Data            []byte         `db:"data"`

	IsProcessed bool `db:"is_processed"`
}

// Convert it to Our Type, with sanity checks
func ParseEthereumLog(l types.Log) AzimuthEventLog {
	event := AzimuthEventLog{
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

func (db *DB) Save(e AzimuthEventLog) {
	fmt.Printf("%#v\n", e)
	_, err := db.DB.NamedExec(`
		insert into event_logs (
			            block_number, block_hash, tx_hash, log_index, contract_address, topic0, topic1, topic2, data, is_processed
			        ) values (
			            :block_number, :block_hash, :tx_hash, :log_index, :contract_address, :topic0, :topic1, :topic2, :data,
			            :is_processed
			        )
	`, e)
	if err != nil {
		panic(err)
	}
}
