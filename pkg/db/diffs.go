package db

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type AzimuthDiff struct {
	ID               uint64        `db:"rowid"`
	SourceEventLogID uint64        `db:"source_event_log_id"`
	IntraLogIndex    uint64        `db:"intra_log_index"` // tx order inside an L2 batch
	AzimuthNumber    AzimuthNumber `db:"azimuth_number"`
	Operation        uint          `db:"operation"`
	Data             []byte        `db:"data"`
}

func (d AzimuthDiff) DataAsUint32() uint32 {
	if len(d.Data) > 4 {
		panic(d.Data)
	}
	ret := uint32(0)
	for _, b := range d.Data {
		ret <<= 8
		ret += uint32(b)
	}
	return ret
}
func (d AzimuthDiff) DataAsAddress() common.Address {
	if len(d.Data) != 20 {
		panic(d.Data)
	}
	return common.BytesToAddress(d.Data)
}
func (d AzimuthDiff) DataAsKeys() (crypto_suite_version uint32, auth_key []byte, encryption_key []byte) {
	if len(d.Data) != 68 {
		panic(d.Data)
	}
	return binary.BigEndian.Uint32(d.Data[:4]), d.Data[4:36], d.Data[36:]
}

func (tx Tx) SaveDiff(d AzimuthDiff) {
	if d.Data == nil {
		d.Data = []byte{}
	}
	_, err := tx.NamedExec(`
		insert into diffs (source_event_log_id, intra_log_index, azimuth_number, operation, data)
		           values (:source_event_log_id, :intra_log_index, :azimuth_number, :operation, :data)`,
		d)
	if err != nil {
		fmt.Printf("%#v\n", d)
		panic(err)
	}
}

const (
	DIFF_SPAWNED = uint(iota + 1)
	DIFF_ACTIVATED
	DIFF_CHANGED_OWNER
	DIFF_CHANGED_SPAWN_PROXY
	DIFF_CHANGED_TRANSFER_PROXY
	DIFF_CHANGED_MANAGEMENT_PROXY
	DIFF_CHANGED_VOTING_PROXY
	DIFF_ESCAPE_REQUESTED
	DIFF_ESCAPE_CANCELED
	DIFF_ESCAPE_ACCEPTED
	DIFF_ESCAPE_REJECTED
	DIFF_LOST_SPONSOR
	DIFF_BREACHED
	DIFF_RESET_KEYS
	DIFF_NEW_DOMINION
)
