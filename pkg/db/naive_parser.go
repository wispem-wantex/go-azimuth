package db

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"
)

const (
	OP_TRANSFER_POINT = iota
	OP_SPAWN
	OP_CONFIGURE_KEYS
	OP_ESCAPE
	OP_CANCEL_ESCAPE
	OP_ADOPT
	OP_REJECT
	OP_DETACH
	OP_SET_MANAGEMENT_PROXY
	OP_SET_SPAWN_PROXY
	OP_SET_TRANSFER_PROXY
)

const (
	PROXY_OWNER = iota
	PROXY_SPAWN
	PROXY_MANAGEMENT
	PROXY_VOTING
	PROXY_TRANSFER
)

type NaiveTx struct {
	// Summary info
	Signature [65]byte
	TxRawData []byte // Includes the whole transaction aside from the signature

	// Who sent the transaction
	SourceShip      AzimuthNumber
	SourceProxyType uint // proxy or owner

	// Transaction details.  General fields which should be sufficient to represent all
	// current transaction types.
	Opcode             uint
	TargetShip         AzimuthNumber
	TargetAddress      common.Address
	Flag               bool
	EncryptionKey      []byte
	AuthKey            []byte
	CryptoSuiteVersion uint
}

// 1. Fetch the source ship+proxy's address and nonce
// 2. Prepare data by concatenating:
//  1. "UrbitIDV1Chain" (14 bytes)
//  2. "1337"           (4 bytes)
//  3. ":"              (1 byte)
//  4. proxy nonce      (4 bytes)
//  5. tx.TxRawData        (unknown bytes)
//
// 3. Prepare signed data by concatenating:   => https://eips.ethereum.org/EIPS/eip-191
//  1. "\x19Ethereum Signed Message:\n" (26 bytes)
//  2. fmt.Sprint(len(prepared_data)) (unknown bytes)
//  3. prepared data from above
//
// 4. Get hash: hash := sha3.NewLegacyKeccak256().Sum(prepared_signed_data)
// 5. Recover public key: pubkey, err := crypto.SigToPub(hash, tx.Signature)
// 6. Derive address from public key: address := crypto.PubkeyToAddress(*pubKey)
// 7. Return address == source ship proxy's address
func (tx NaiveTx) VerifySignature(source_ship_point Point) bool {
	var eth_chain_id = []byte("1") // Ethereum Mainnet chain ID
	var urbit_chain_id = []byte("UrbitIDV1Chain")

	// Get the appropriate proxy address and nonce
	var proxy_address common.Address
	var proxy_nonce uint32
	switch tx.SourceProxyType {
	case PROXY_OWNER:
		proxy_address = source_ship_point.OwnerAddress
		proxy_nonce = source_ship_point.OwnerNonce
	case PROXY_SPAWN:
		proxy_address = source_ship_point.SpawnAddress
		proxy_nonce = source_ship_point.SpawnNonce
	case PROXY_MANAGEMENT:
		proxy_address = source_ship_point.ManagementAddress
		proxy_nonce = source_ship_point.ManagementNonce
	case PROXY_VOTING:
		proxy_address = source_ship_point.VotingAddress
		proxy_nonce = source_ship_point.VotingNonce
	case PROXY_TRANSFER:
		proxy_address = source_ship_point.TransferAddress
		proxy_nonce = source_ship_point.TransferNonce
	}
	nonce_bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(nonce_bytes, proxy_nonce)

	reverse := func(b []byte) []byte {
		newSlice := make([]byte, len(b))
		copy(newSlice, b)

		for i, j := 0, len(newSlice)-1; i < j; i, j = i+1, j-1 {
			newSlice[i], newSlice[j] = newSlice[j], newSlice[i]
		}

		return newSlice
	}

	// Prepare data with UrbitID salt, Ethereum chain ID, nonce, and `personal_sign` armoring.
	// `personal_sign`: https://eips.ethereum.org/EIPS/eip-191
	prepared_data := urbit_chain_id
	prepared_data = append(prepared_data, eth_chain_id...)
	prepared_data = append(prepared_data, []byte(":")...)
	prepared_data = append(prepared_data, nonce_bytes...)
	prepared_data = append(prepared_data, reverse(tx.TxRawData)...)

	// WTF: the "byte length" is the *string-encoded* base-10 length of the prepared data
	// Apparently the message has to be valid ASCII, otherwise some wallets won't accept it for
	// signing; see https://x.com/pcmonk/status/1842341477604802700
	byte_length := len(prepared_data)
	byte_length_bytes := []byte(fmt.Sprint(byte_length)) // Insane but oh well

	// Assemble the actual signed message
	signed_data := []byte("\x19Ethereum Signed Message:\n")
	signed_data = append(signed_data, byte_length_bytes...)
	signed_data = append(signed_data, prepared_data...)

	// fmt.Println("------------------")
	// fmt.Printf("Prepared data: %x\n", prepared_data)
	// fmt.Printf("Signed data: %x\n", signed_data)
	// fmt.Printf("Signed data (reversed): %x\n", reverse(signed_data))

	// WTF: Fix "v". `crypto.SigToPub` expects it to be "0" or "1", but it might have +27 added
	// for some reason, and `go-ethereum/crypto` doesn't handle this!
	if tx.Signature[len(tx.Signature)-1] >= 27 {
		tx.Signature[len(tx.Signature)-1] = tx.Signature[len(tx.Signature)-1] - 27
	}

	// Hash it
	hash := sha3.NewLegacyKeccak256()
	hash.Write(signed_data)
	// fmt.Printf("Hash: %x\n", hash.Sum(nil))
	// fmt.Printf("Signature: %x\n", tx.Signature[:])

	// Recover the address from signed message and signature
	pubkey, err := crypto.SigToPub(hash.Sum(nil), tx.Signature[:])
	if err != nil {
		panic(err)
	}
	address := crypto.PubkeyToAddress(*pubkey)
	// fmt.Println(address)
	return address == proxy_address
}

// 1. Reverse the byte slice
// 2. 65 bytes => signature
// 3. 1 byte => proxy type (incl. owner)
// 4. 4 bytes => ship
// 5. 7 bites => opcode
// 6. 1 bit => misc (flags or padding)
//
// Opcode:
//   - 0 (transfer point):
//     flag => "reset" (0 = true, 1 = false)
//     20 bytes => eth_address
//   - 1 (spawn):
//     4 bytes => ship
//     20 bytes => eth_address
//   - 2 (configure keys)
//     flag => "breach" (0 = true, 1 = false)
//     32 bytes => encryption key
//     32 bytes => auth key
//     4 bytes => crypto-suite
//   - 3 (escape)
//     4 bytes => ship
//   - 4 (cancel-escape)
//     4 bytes => ship
//   - 5 (adopt)
//     4 bytes => ship
//   - 6 (reject)
//     4 bytes => ship
//   - 7 (detach)
//     4 bytes => ship
//   - 8 (set management proxy)
//     20 bytes => eth_address
//   - 9 (set spawn proxy)
//     20 bytes => eth_address
//   - 10 (set transfer proxy)
//     20 bytes => eth_address
func ParseNaiveBatch(batch []byte) []NaiveTx {
	ret := []NaiveTx{}

	i := int(len(batch))
	var j int
	for i > 0 {
		tx := NaiveTx{}

		i, j = i-65, i
		copy(tx.Signature[:65], batch[i:j])
		tx_mark := i // Set a mark so after parsing we can set the whole TxRawData bytes

		i, _ = i-1, i
		tx.SourceProxyType = uint(batch[i])

		i, j = i-4, i
		tx.SourceShip = AzimuthNumber(binary.BigEndian.Uint32(batch[i:j]))
		// fmt.Printf("Ship: ~%s\n", phonemes.IntToPhoneme(ship))

		i, _ = i-1, i
		tx.Opcode = 0x7f & uint(batch[i])
		flag := (batch[i] >> 7) == 0
		switch tx.Opcode {
		case OP_TRANSFER_POINT:
			i, j = i-20, i
			tx.TargetAddress = common.BytesToAddress(batch[i:j])
			tx.Flag = flag // reset
		case OP_SPAWN:
			i, j = i-4, i
			tx.TargetShip = AzimuthNumber(binary.BigEndian.Uint32(batch[i:j]))
			i, j = i-20, i
			tx.TargetAddress = common.BytesToAddress(batch[i:j])
		case OP_CONFIGURE_KEYS:
			i, j = i-32, i
			tx.EncryptionKey = batch[i:j]

			i, j = i-32, i
			tx.AuthKey = batch[i:j]

			// WTF: if this is the last tx in the batch, leading "0x00"s could be chopped off
			i, _ = i-4, i //
			tx.CryptoSuiteVersion = uint(batch[max(0, i)])
			tx.Flag = flag // breach
		case OP_ESCAPE:
			i, j = i-4, i
			tx.TargetShip = AzimuthNumber(binary.BigEndian.Uint32(batch[i:j]))
		case OP_CANCEL_ESCAPE:
			i, j = i-4, i
			tx.TargetShip = AzimuthNumber(binary.BigEndian.Uint32(batch[i:j]))
		case OP_ADOPT:
			i, j = i-4, i
			tx.TargetShip = AzimuthNumber(binary.BigEndian.Uint32(batch[i:j]))
		case OP_REJECT:
			i, j = i-4, i
			tx.TargetShip = AzimuthNumber(binary.BigEndian.Uint32(batch[i:j]))
		case OP_DETACH:
			i, j = i-4, i
			tx.TargetShip = AzimuthNumber(binary.BigEndian.Uint32(batch[i:j]))
		case OP_SET_MANAGEMENT_PROXY:
			i, j = i-20, i
			tx.TargetAddress = common.BytesToAddress(batch[i:j])
		case OP_SET_SPAWN_PROXY:
			i, j = i-20, i
			tx.TargetAddress = common.BytesToAddress(batch[i:j])
		case OP_SET_TRANSFER_PROXY:
			i, j = i-20, i
			tx.TargetAddress = common.BytesToAddress(batch[i:j])
		}

		// Handling for CryptoSuiteVersion being unpadded for some reason
		padding_length := -min(0, i) // If `i` is negative, add `-i` bytes of padding
		tx.TxRawData = append(make([]byte, padding_length), batch[max(0, i):tx_mark]...)
		ret = append(ret, tx)
	}
	return ret
}
