package db_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "go-azimuth/pkg/db"
)

func TestRiftEvent(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	event := EthereumEventLog{
		TxHash:          common.HexToHash("4a085416f030d5a84ada63d08bd96711b2765afc0590e15efbdeff44e03aad63"),
		LogIndex:        190,
		ContractAddress: common.HexToAddress("223c067f8cf28ae173ee5cafea60ca44c335fecb"),
		Name:            "BrokeContinuity",
		Topic0:          common.HexToHash("29294799f1c21a37ef838e15f79dd91bcee2df99d63cd1c18ac968b129514e6e"),
		Topic1:          common.HexToHash("000000000000000000000000000000000000000000000000000000005ae3aca0"),
		Topic2:          common.HexToHash("0000000000000000000000000000000000000000000000000000000000000000"),
		Data:            hex_to_bytes("0000000000000000000000000000000000000000000000000000000000000001"),
	}

	azm_num := AzimuthNumber(1524870304)
	db, err := DBCreate(":memory:")
	assert.NoError(err)
	db.DB.MustExec(`insert into points (azimuth_number, dominion) values (?, 1)`, azm_num)
	tx := Tx{db.DB.MustBegin()}

	q, diffs := event.Effects(tx)
	p, is_ok := q.BindValues.(Point)
	require.True(is_ok)
	assert.Equal(uint32(1), p.Rift)
	assert.Equal(azm_num, p.Number)
	require.Len(diffs, 1)
	assert.Equal(DIFF_BREACHED, diffs[0].Operation)
	assert.Equal(azm_num, diffs[0].AzimuthNumber)
	assert.Equal([]byte{0x0, 0x0, 0x0, 0x1}, diffs[0].Data)
}
