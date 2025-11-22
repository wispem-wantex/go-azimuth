package db

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
)

type AzimuthRank uint

const (
	GALAXY = AzimuthRank(iota)
	STAR
	PLANET
	MOON
	COMET
)

type AzimuthNumber uint32

// Get the natural parent of an Azimuth point.
func (p AzimuthNumber) Parent() AzimuthNumber {
	if p > 0xffff {
		// Planets' parent is their star
		return p & 0xffff
	} else if p > 0xff {
		// Stars' parent is their galaxy
		return p & 0xff
	}
	// Galaxies don't have a parent
	return 0
}

// Get the "rank" of an Azimuth point
func (p AzimuthNumber) Rank() AzimuthRank {
	if p <= 0xff {
		return GALAXY
	} else if p <= 0xffff {
		return STAR
	} else {
		return PLANET
	}
}

type Point struct {
	Number AzimuthNumber `db:"azimuth_number"`

	// Nonces are for L2 to prevent replay attacks
	OwnerAddress      common.Address `db:"owner_address"`
	OwnerNonce        uint32         `db:"owner_nonce"`
	SpawnAddress      common.Address `db:"spawn_address"`
	SpawnNonce        uint32         `db:"spawn_nonce"`
	ManagementAddress common.Address `db:"management_address"`
	ManagementNonce   uint32         `db:"management_nonce"`
	VotingAddress     common.Address `db:"voting_address"`
	VotingNonce       uint32         `db:"voting_nonce"`
	TransferAddress   common.Address `db:"transfer_address"`
	TransferNonce     uint32         `db:"transfer_nonce"`

	Dominion int    `db:"dominion"`
	IsActive bool   `db:"is_active"`
	Rift     uint32 `db:"rift"`

	EncryptionKey      []byte `db:"encryption_key"`
	AuthKey            []byte `db:"auth_key"`
	CryptoSuiteVersion uint32 `db:"crypto_suite_version"`
	Life               uint32 `db:"life"`

	HasSponsor        bool          `db:"has_sponsor"`
	Sponsor           AzimuthNumber `db:"sponsor"`
	IsEscapeRequested bool          `db:"is_escape_requested"`
	EscapeRequestedTo AzimuthNumber `db:"escape_requested_to"`
}

func (p Point) MarshalJSON() ([]byte, error) {
	type Alias Point

	result, err := json.Marshal(&struct {
		EncryptionKey string `json:"EncryptionKey"`
		AuthKey       string `json:"AuthKey"`
		Alias
	}{
		EncryptionKey: hex.EncodeToString(p.EncryptionKey),
		AuthKey:       hex.EncodeToString(p.AuthKey),
		Alias:         (Alias)(p),
	})
	if err != nil {
		err = fmt.Errorf("encoding json: %w", err)
	}
	return result, err
}

func (db DB) GetPoint(azimuth_number AzimuthNumber) (Point, bool) {
	var ret Point
	err := db.DB.Get(&ret, `select * from points where azimuth_number = ?`, azimuth_number)
	if errors.Is(err, sql.ErrNoRows) {
		return Point{}, false
	} else if err != nil {
		panic(err)
	}
	return ret, true
}

type PointHistory struct {
	ID               uint64        `db:"rowid"`
	ContractName     string        `db:"contract"`
	TxHash           string        `db:"tx_hash"`
	SourceEventLogID uint64        `db:"source_event_log_id"`
	IntraLogIndex    uint64        `db:"intra_log_index"`
	AzimuthNumber    AzimuthNumber `db:"azimuth_number"`
	OperationName    string        `db:"operation"`
	HexData          string        `db:"hex_data"`
}

func (db DB) GetEventsForPoint(azimuth_number AzimuthNumber) (ret []PointHistory, is_ok bool) {
	err := db.DB.Select(&ret, `select * from readable_diffs where azimuth_number = ?`, azimuth_number)
	if errors.Is(err, sql.ErrNoRows) {
		return []PointHistory{}, false
	} else if err != nil {
		panic(err)
	}
	return ret, true
}
