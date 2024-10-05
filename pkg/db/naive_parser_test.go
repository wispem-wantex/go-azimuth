package db_test

import (
	"encoding/hex"
	// "fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	. "go-azimuth/pkg/db"
)

func hex_to_bytes(s string) []byte {
	data, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return data
}
func hex_to_signature(s string) [65]byte {
	data := hex_to_bytes(s)
	if len(data) != 65 {
		panic(len(data))
	}
	var ret [65]byte
	copy(ret[:], data[:65])
	return ret
}

func TestParseNaiveBatch(t *testing.T) {
	assert := assert.New(t)
	test_cases := []struct {
		Data   []byte
		Result []NaiveTx
	}{
		{
			hex_to_bytes(
				"671738dada5c209c12b6501e80c62e091c27b14a8022d601a20473fdca35f685fa61153a73bef738eb" +
					"fbf4cf95cca253dd39343ad3ae287e156228693b06f7a66defb761109e1d3c3bc5be348c28b22ae272" +
					"d83709ea8a9acf031c671738dada5c209c12b6501e80c62e091c27b14a0a22d601a2000ef750117707" +
					"57f561b40f3ba2dea676af739795101800457a19b68698bbdfde653437558121965eef535c95f96780" +
					"1c4e3a7928cb9d06a6fa66b4e97ca43e9500"),
			[]NaiveTx{
				{
					Signature: hex_to_signature(
						"0ef75011770757f561b40f3ba2dea676af739795101800457a19b68698bbdfde6534375581" +
							"21965eef535c95f967801c4e3a7928cb9d06a6fa66b4e97ca43e9500"),
					TxRawData:       hex_to_bytes("671738dada5c209c12b6501e80c62e091c27b14a0a22d601a200"),
					SourceShip:      AzimuthNumber(584450466),
					SourceProxyType: PROXY_OWNER,
					Opcode:          OP_SET_TRANSFER_PROXY,
					TargetAddress:   common.Address(hex_to_bytes("671738dada5c209c12b6501e80c62e091c27b14a")),
				}, {
					Signature: hex_to_signature("73fdca35f685fa61153a73bef738ebfbf4cf95cca253" +
						"dd39343ad3ae287e156228693b06f7a66defb761109e1d3c3bc5be348c28b22ae272d83709ea8a9acf031c"),
					TxRawData:       hex_to_bytes("671738dada5c209c12b6501e80c62e091c27b14a8022d601a204"),
					SourceShip:      AzimuthNumber(584450466),
					SourceProxyType: PROXY_TRANSFER,
					Opcode:          OP_TRANSFER_POINT,
					TargetAddress:   common.Address(hex_to_bytes("671738dada5c209c12b6501e80c62e091c27b14a")),
					Flag:            false, // reset flag
				},
			},
		},
		{
			hex_to_bytes(
				"01f9900aa356eb818275c9bc58c355d075570094503a01a510270c78f30724fd7ef387f5c96dad3a56" +
					"5e78dcfda556e4d36a8257e187d7106ea5ecabd2f6b5fd82820342050000566725a9b02227bfc58d82" +
					"218061a28d17d750b32e1828b7dc0d32675431a27e4d50b943315d9dcaf335e0a1c4418be025df9347" +
					"86f937e7e973d22e07d4e60601"),
			[]NaiveTx{
				{
					Signature: hex_to_signature(
						"566725a9b02227bfc58d82218061a28d17d750b32e1828b7dc0d32675431a27e4d50b943315d9dc" +
							"af335e0a1c4418be025df934786f937e7e973d22e07d4e60601"),
					TxRawData: hex_to_bytes(
						"00000001f9900aa356eb818275c9bc58c355d075570094503a01a510270c78f30724fd7ef387f5c" +
							"96dad3a565e78dcfda556e4d36a8257e187d7106ea5ecabd2f6b5fd82820342050000"),
					SourceShip:         AzimuthNumber(54658304),
					SourceProxyType:    PROXY_OWNER,
					Opcode:             OP_CONFIGURE_KEYS,
					EncryptionKey:      hex_to_bytes("f387f5c96dad3a565e78dcfda556e4d36a8257e187d7106ea5ecabd2f6b5fd82"),
					AuthKey:            hex_to_bytes("f9900aa356eb818275c9bc58c355d075570094503a01a510270c78f30724fd7e"),
					CryptoSuiteVersion: 1,
					Flag:               false, // breach
				},
			},
		},
	}

	for _, tc := range test_cases {
		rslt := ParseNaiveBatch(tc.Data)
		for i := range tc.Result {
			assert.Equal(tc.Result[i], rslt[i])
		}
	}
}

func TestCheckNaiveSignatures(t *testing.T) {
	assert := assert.New(t)
	test_cases := []struct {
		Tx     NaiveTx
		Sender Point
	}{
		{
			Tx: NaiveTx{
				Signature: hex_to_signature(
					"0ef75011770757f561b40f3ba2dea676af739795101800457a19b68698bbdfde65343755812196" +
						"5eef535c95f967801c4e3a7928cb9d06a6fa66b4e97ca43e9500"),
				TxRawData:       hex_to_bytes("671738dada5c209c12b6501e80c62e091c27b14a0a22d601a200"),
				SourceShip:      AzimuthNumber(584450466),
				SourceProxyType: PROXY_OWNER,
				Opcode:          OP_SET_TRANSFER_PROXY,
				TargetAddress:   common.Address(hex_to_bytes("671738dada5c209c12b6501e80c62e091c27b14a")),
			},
			Sender: Point{
				OwnerAddress: common.HexToAddress("942cc0b03f531bb7359347c4f272babb2eaf0c99"),
			},
		},
		{
			Tx: NaiveTx{
				Signature: hex_to_signature(
					"73fdca35f685fa61153a73bef738ebfbf4cf95cca253dd39343ad3ae287e156228693b06f7a66d" +
						"efb761109e1d3c3bc5be348c28b22ae272d83709ea8a9acf031c"),
				TxRawData:       hex_to_bytes("671738dada5c209c12b6501e80c62e091c27b14a8022d601a204"),
				SourceShip:      AzimuthNumber(584450466),
				SourceProxyType: PROXY_TRANSFER,
				Opcode:          OP_TRANSFER_POINT,
				TargetAddress:   common.Address(hex_to_bytes("671738dada5c209c12b6501e80c62e091c27b14a")),
				Flag:            false, // reset flag
			},
			Sender: Point{
				TransferAddress: common.HexToAddress("671738dada5c209c12b6501e80c62e091c27b14a"),
			},
		},
		{
			Tx: NaiveTx{
				Signature: hex_to_signature(
					"566725a9b02227bfc58d82218061a28d17d750b32e1828b7dc0d32675431a27e4d50b943315d9d" +
						"caf335e0a1c4418be025df934786f937e7e973d22e07d4e60601"),
				TxRawData: hex_to_bytes(
					"00000001f9900aa356eb818275c9bc58c355d075570094503a01a510270c78f30724fd7ef387f5" +
						"c96dad3a565e78dcfda556e4d36a8257e187d7106ea5ecabd2f6b5fd82820342050000"),
				SourceShip:         AzimuthNumber(54658304),
				SourceProxyType:    PROXY_OWNER,
				Opcode:             OP_CONFIGURE_KEYS,
				EncryptionKey:      hex_to_bytes("f387f5c96dad3a565e78dcfda556e4d36a8257e187d7106ea5ecabd2f6b5fd82"),
				AuthKey:            hex_to_bytes("f9900aa356eb818275c9bc58c355d075570094503a01a510270c78f30724fd7e"),
				CryptoSuiteVersion: 1,
				Flag:               false, // breach
			},
			Sender: Point{
				OwnerAddress: common.HexToAddress("46b24384f42c324e71b60e7a5c24ad20ee6faa20"),
			},
		},
	}
	for _, tc := range test_cases {
		rslt := tc.Tx.VerifySignature(tc.Sender)
		assert.True(rslt)
	}
}
