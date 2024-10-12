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
					IntraLogIndex: 1,
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
		{
			// Check handling of WTF-truncated padding on OP_ESCAPE operations
			hex_to_bytes(
				"3eb303beccfeac00594a127b1ea1a880de4c446971932d15b1b021ad13be5e09f98ebb71f45f06967f17bac" +
					"ca27d84b448ec5809c15ca48ff73d47a541e94be994ef774fe9b56ba01b00003eb30321cdfeac00845adad" +
					"fb162528b3df3d407b8177060b5284f9d9257cee641b143e4cfd30af74ca01325ef50be6349d4ae6969899" +
					"2fcf74ef544e63ae1461d7ce1913000a3ef1c"),
			[]NaiveTx{
				{
					Signature: hex_to_signature(
						"845adadfb162528b3df3d407b8177060b5284f9d9257cee641b143e4cfd30af74ca01325ef50be634" +
							"9d4ae69698992fcf74ef544e63ae1461d7ce1913000a3ef1c"),
					TxRawData:       hex_to_bytes("00003eb30321cdfeac00"),
					SourceShip:      AzimuthNumber(567148204),
					SourceProxyType: PROXY_OWNER,
					Opcode:          OP_ESCAPE,
					TargetShip:      AzimuthNumber(16051),
				},
				{
					IntraLogIndex: 1,
					Signature: hex_to_signature(
						"594a127b1ea1a880de4c446971932d15b1b021ad13be5e09f98ebb71f45f06967f17bacca27d84b448" +
							"ec5809c15ca48ff73d47a541e94be994ef774fe9b56ba01b"),
					TxRawData:       hex_to_bytes("00003eb303beccfeac00"),
					SourceShip:      AzimuthNumber(3201105580),
					SourceProxyType: PROXY_OWNER,
					Opcode:          OP_ESCAPE,
					TargetShip:      AzimuthNumber(16051),
				},
			},
		},
		{
			// idk, another test case I suppose
			hex_to_bytes("661803651AC018009399A8A143B21FCA343109128370BD9575B17F97BE491985D4932F309B4088BA5B" +
				"9E2666ED5FF6322B0594AD31F6BD966B7D5A0B55738C43FA2E80517858F27C00"),
			[]NaiveTx{
				{
					Signature: hex_to_signature("9399a8a143b21fca343109128370bd9575b17f97be491985d4932" +
						"f309b4088ba5b9e2666ed5ff6322b0594ad31f6bd966b7d5a0b55738c43fa2e80517858f27c00"),
					TxRawData:       hex_to_bytes("0000661803651ac01800"),
					SourceShip:      AzimuthNumber(1696251928),
					SourceProxyType: PROXY_OWNER,
					Opcode:          OP_ESCAPE,
					TargetShip:      AzimuthNumber(26136),
				},
			},
		},
		{
			// Crypto suite versions-- should not be 0
			hex_to_bytes("000000010A56E0A92352BF6723E518122DCE95A0EAAFBA3B3028B8806D5E7AD6AC56FF1E6A753FC18CC" +
				"1BA858A4F344A164098247B607482812D6D5F75A8987EF62F9F3D82651AC018004962588F587083D2044965626440C" +
				"B2281FE6DD413CF4F9EA90587BB98444DE5473DFF657EB3B0D6E03B67FA6367C483543AD7C2C8B678FF96A05E071B2F680200"),
			[]NaiveTx{
				{
					Signature: hex_to_signature("4962588f587083d2044965626440cb2281fe6dd413cf4f9ea90587" +
						"bb98444de5473dff657eb3b0d6e03b67fa6367c483543ad7c2c8b678ff96a05e071b2f680200"),
					TxRawData: hex_to_bytes("000000010a56e0a92352bf6723e518122dce95a0eaafba3b3028b8806d" +
						"5e7ad6ac56ff1e6a753fc18cc1ba858a4f344a164098247b607482812d6d5f75a8987ef62f9f3d82651ac01800"),
					SourceShip:         AzimuthNumber(1696251928),
					SourceProxyType:    PROXY_OWNER,
					Opcode:             OP_CONFIGURE_KEYS,
					EncryptionKey:      hex_to_bytes("6a753fc18cc1ba858a4f344a164098247b607482812d6d5f75a8987ef62f9f3d"),
					AuthKey:            hex_to_bytes("0a56e0a92352bf6723e518122dce95a0eaafba3b3028b8806d5e7ad6ac56ff1e"),
					CryptoSuiteVersion: 1,
				},
			},
		},
	}

	for _, tc := range test_cases {
		rslt := ParseNaiveBatch(tc.Data, 0)
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
		{
			// Signature with non-zero nonce
			Tx: NaiveTx{
				Signature: hex_to_signature(
					"cc9dba6a6c90616dac53c1cff8e224e02ce9601fcec85c68f73a77bb5970354a523842d5335d847" +
						"f0f76c95b7cbb735f0847b0e22c9040da1dd1ed3ac4c2b09900"),
				TxRawData:       hex_to_bytes("bad132bf9c1269ee4fbc3affc537d3de3adb169b0001fbea010000fbea00"),
				SourceShip:      AzimuthNumber(64490),
				SourceProxyType: PROXY_OWNER,
				Opcode:          OP_SPAWN,
				TargetShip:      AzimuthNumber(130026),
			},
			Sender: Point{
				OwnerAddress: common.HexToAddress("baD132Bf9c1269ee4fBc3AFfC537d3De3Adb169B"),
				OwnerNonce:   1,
			},
		},
	}
	for _, tc := range test_cases {
		rslt := tc.Tx.VerifySignature(tc.Sender)
		assert.True(rslt)
	}
}
