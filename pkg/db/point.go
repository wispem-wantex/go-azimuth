package db

import (
	"database/sql"
	"errors"
	"github.com/ethereum/go-ethereum/common"
)

type AzimuthNumber uint32

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
