package db

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"errors"
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
	EthereumEventLogID uint64
	IntraLogIndex      uint64
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
	CryptoSuiteVersion uint32
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

// Play all events (both azimuth and naive) since the start of the Naive contract.
//
// WTF(naive-azimuth-interlacing): Naive does not actually "happen after" Azimuth.  For example, consider the
// following scenario.  An L1 star can sponsor an L2 planet.  After using a certain proxy to adopt
// an escaping L2 planet (L2 tx), the L1 star could change their proxy address on L1.  The L2 tx
// is valid, but if it was processed after all L1 txs, the processing would see the new proxy
// address and incorrectly consider the L2 tx invalid.
//
// So L1 and L2 txs have to actually be processed in order, interleaving between the two.
func (db *DB) PlayNaiveLogs() {
	var events []EthereumEventLog
	for {
		err := db.DB.Select(&events, `
		    select rowid, block_number, block_hash, tx_hash, log_index, contract_address, topic0, topic1,
		            topic2, data, is_processed from ethereum_events
		     where is_processed = 0
		  order by block_number, log_index asc
		`)
		if err != nil {
			panic(err)
		} else if len(events) == 0 {
			// No unprocessed logs left; we're finished
			break
		}
		for _, e := range events {
			if e.ContractAddress == common.HexToAddress("eb70029cfb3c53c778eaf68cd28de725390a1fe9") {
				// Naive
				db.ApplyBatchEvent(e)
			} else {
				// Azimuth
				db.ApplyEventEffects([]EthereumEventLog{e})
			}
		}
	}
}

func (db *DB) ApplyBatchEvent(event EthereumEventLog) {
	if event.Topic0 != BATCH {
		panic(event)
	}

	naive_txs := ParseNaiveBatch(event.Data, event.ID)
	for _, tx := range naive_txs {
		var p Point
		err := db.DB.Get(&p, `select * from points where azimuth_number = ?`, tx.SourceShip)
		if err != nil {
			panic(err)
		}

		// Check signature
		if !tx.VerifySignature(p) {
			fmt.Printf("\n>>>   Signature failed to verify in batch (%d, %d): %#v\n", event.BlockNumber, event.LogIndex, tx)
			continue
		}

		// Get effects
		effects, diffs := tx.Effects(db)
		for _, q := range effects {
			_, err = db.DB.NamedExec(q.SQL, q.BindValues)
			if err != nil {
				fmt.Printf("%q; %#v\n", q.SQL, q.BindValues)
				panic(err)
			}
		}

		for _, d := range diffs {
			db.SaveDiff(d)
		}
	}
	_, err := db.DB.NamedExec(`
		update ethereum_events
		   set is_processed=1
		 where block_number = :block_number and log_index = :log_index`,
		event)
	if err != nil {
		panic(err)
	}
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
func ParseNaiveBatch(batch []byte, ethereum_event_log_id uint64) []NaiveTx {
	ret := []NaiveTx{}

	uint32_from_bytes := func(b []byte) uint32 {
		padded_bytes := make([]byte, 4)
		copy(padded_bytes[4-len(b):], b) // Right-align the bytes in a 4-byte array
		return binary.BigEndian.Uint32(padded_bytes)
	}

	intra_log_index := uint64(0)

	i := int(len(batch))
	var j int
	for i > 0 {
		tx := NaiveTx{EthereumEventLogID: ethereum_event_log_id, IntraLogIndex: intra_log_index}

		i, j = i-65, i
		copy(tx.Signature[:65], batch[max(0, i):j])
		tx_mark := i // Set a mark so after parsing we can set the whole TxRawData bytes

		i, _ = i-1, i
		tx.SourceProxyType = uint(batch[max(0, i)])

		i, j = i-4, i
		tx.SourceShip = AzimuthNumber(binary.BigEndian.Uint32(batch[max(0, i):j]))
		// fmt.Printf("Ship: ~%s\n", phonemes.IntToPhoneme(ship))

		i, _ = i-1, i
		tx.Opcode = 0x7f & uint(batch[max(0, i)])
		flag := (batch[max(0, i)] >> 7) == 0
		switch tx.Opcode {
		case OP_TRANSFER_POINT:
			i, j = i-20, i
			tx.TargetAddress = common.BytesToAddress(batch[max(0, i):j])
			tx.Flag = flag // reset
		case OP_SPAWN:
			i, j = i-4, i
			tx.TargetShip = AzimuthNumber(binary.BigEndian.Uint32(batch[max(0, i):j]))
			i, j = i-20, i
			tx.TargetAddress = common.BytesToAddress(batch[max(0, i):j])
		case OP_CONFIGURE_KEYS:
			i, j = i-32, i
			tx.EncryptionKey = batch[max(0, i):j]

			i, j = i-32, i
			tx.AuthKey = batch[max(0, i):j]

			i, j = i-4, i //
			tx.CryptoSuiteVersion = uint32_from_bytes(batch[max(0, i):j])
			tx.Flag = flag // breach
		case OP_ESCAPE:
			i, j = i-4, i
			tx.TargetShip = AzimuthNumber(uint32_from_bytes(batch[max(0, i):j]))
		case OP_CANCEL_ESCAPE:
			i, j = i-4, i
			tx.TargetShip = AzimuthNumber(uint32_from_bytes(batch[max(0, i):j]))
		case OP_ADOPT:
			i, j = i-4, i
			tx.TargetShip = AzimuthNumber(uint32_from_bytes(batch[max(0, i):j]))
		case OP_REJECT:
			i, j = i-4, i
			tx.TargetShip = AzimuthNumber(uint32_from_bytes(batch[max(0, i):j]))
		case OP_DETACH:
			i, j = i-4, i
			tx.TargetShip = AzimuthNumber(uint32_from_bytes(batch[max(0, i):j]))
		case OP_SET_MANAGEMENT_PROXY:
			i, j = i-20, i
			tx.TargetAddress = common.BytesToAddress(batch[max(0, i):j])
		case OP_SET_SPAWN_PROXY:
			i, j = i-20, i
			tx.TargetAddress = common.BytesToAddress(batch[max(0, i):j])
		case OP_SET_TRANSFER_PROXY:
			i, j = i-20, i
			tx.TargetAddress = common.BytesToAddress(batch[max(0, i):j])
		}

		// WTF: if this is the last tx in the batch, leading "0x00"s could be omitted from the
		// batch data, but they have to be added back in for the signatures to validate!
		// This is because the Hoon implementation processes batch data as atoms, which have
		// infinite (implicit) preceeding zeroes.
		padding_length := -min(0, i) // If `i` is negative, add `-i` bytes of padding
		tx.TxRawData = append(make([]byte, padding_length), batch[max(0, i):tx_mark]...)
		ret = append(ret, tx)
		intra_log_index += 1
	}
	return ret
}

func (tx NaiveTx) Effects(db *DB) ([]Query, []AzimuthDiff) {
	// helper func
	get_point := func(n AzimuthNumber) (ret Point) {
		err := db.DB.Get(&ret, `select * from points where azimuth_number = ?`, n)
		if err != nil {
			panic(err)
		}
		return ret
	}
	p := get_point(tx.SourceShip)

	ret := []Query{}
	diffs := []AzimuthDiff{}

	// Increment the appropriate nonce, even if the transaction doesn't apply properly
	// If the raw-tx parses properly, then we want to avoid people re-broadcasting it
	increment_nonce_query := Query{BindValues: p}
	switch tx.SourceProxyType {
	case PROXY_OWNER:
		increment_nonce_query.SQL = "update points set owner_nonce=owner_nonce+1 where azimuth_number = :azimuth_number"
	case PROXY_SPAWN:
		increment_nonce_query.SQL = "update points set spawn_nonce=spawn_nonce+1 where azimuth_number = :azimuth_number"
	case PROXY_MANAGEMENT:
		increment_nonce_query.SQL = "update points set management_nonce=management_nonce+1 where azimuth_number = :azimuth_number"
	case PROXY_VOTING:
		increment_nonce_query.SQL = "update points set voting_nonce=voting_nonce+1 where azimuth_number = :azimuth_number"
	case PROXY_TRANSFER:
		increment_nonce_query.SQL = "update points set transfer_nonce=transfer_nonce+1 where azimuth_number = :azimuth_number"
	}
	ret = append(ret, increment_nonce_query)

	// Apply the transaction.
	// We have to do a lot more validation here than on L1 since there's no smart contract to make
	// guarantees for us.
	switch tx.Opcode {
	case OP_TRANSFER_POINT:
		// 1. Assert SourceShip is on L2
		// 2. Assert SourceProxyType is permitted, either "owner" or "transfer proxy"
		if p.Dominion != 2 {
			fmt.Printf("Ignoring tx: source is not an L2 ship\n")
			break
		}
		if tx.SourceProxyType != PROXY_OWNER && tx.SourceProxyType != PROXY_TRANSFER {
			fmt.Printf("Ignoring tx: source proxy is not authorized to transfer\n")
			break
		}

		// 3. Update owner address; zero out transfer address
		p.OwnerAddress = tx.TargetAddress
		p.TransferAddress = common.Address{}
		diffs = append(diffs,
			AzimuthDiff{
				SourceEventLogID: tx.EthereumEventLogID,
				IntraLogIndex:    tx.IntraLogIndex,
				AzimuthNumber:    p.Number,
				Operation:        DIFF_CHANGED_OWNER,
				Data:             p.OwnerAddress[:],
			})

		// 4. If reset is requested
		if tx.Flag {
			// 1. If p.Life is not 0: increment p.Rift (out of order in naive.hoon, for simplicity)
			if p.Life != 0 {
				p.Rift += 1
				diffs = append(diffs,
					AzimuthDiff{
						SourceEventLogID: tx.EthereumEventLogID,
						IntraLogIndex:    tx.IntraLogIndex,
						AzimuthNumber:    p.Number,
						Operation:        DIFF_BREACHED,
					})
			}
			// 2. if p.Suite, p.Auth and p.Encrypt are not all 0, set them to 0x0000...0000 and increment p.Life
			if p.CryptoSuiteVersion != 0 ||
				!bytes.Equal(p.AuthKey, make([]byte, len(p.AuthKey))) ||
				!bytes.Equal(p.EncryptionKey, make([]byte, len(p.EncryptionKey))) {
				p.Life += 1
				diffs = append(diffs,
					AzimuthDiff{
						SourceEventLogID: tx.EthereumEventLogID,
						IntraLogIndex:    tx.IntraLogIndex,
						AzimuthNumber:    p.Number,
						Operation:        DIFF_RESET_KEYS,
						Data:             []byte{},
					})
			}
			p.CryptoSuiteVersion = 0
			p.AuthKey = []byte{}
			p.EncryptionKey = []byte{}
			// 3. Set p.SpawnAddress, p.ManagementAddress, p.VotingAddress and p.TransferAddress = 0x0000...0000
			p.SpawnAddress = common.Address{}
			p.ManagementAddress = common.Address{}
			p.VotingAddress = common.Address{}
			p.TransferAddress = common.Address{}
		}

		// 5. Save the result
		ret = append(ret,
			Query{`
				update points
				   set owner_address = :owner_address,
				       transfer_address = :transfer_address,
				       life = :life,
				       rift = :rift,
				       crypto_suite_version = :crypto_suite_version,
				       auth_key = :auth_key,
				       encryption_key = :encryption_key,
				       spawn_address = :spawn_address,
				       management_address = :management_address,
				       voting_address = :voting_address,
				       transfer_address = :transfer_address
				 where azimuth_number = :azimuth_number`,
				p,
			})

	case OP_SPAWN:
		// TargetShip is the ship getting spawned; TargetAddress is who will be the new owner
		// 3. Assert the transaction's SourceShip (sender) is the natural parent of TargetShip
		if tx.SourceShip != tx.TargetShip.Parent() {
			fmt.Printf("Ignoring tx: source is not target's parent to spawn it\n")
			break
		}
		// 2. Assert tx.SourceShip is on L2 or Spawn dominion
		if p.Dominion != 2 && p.Dominion != 3 {
			fmt.Printf("Ignoring tx: source ship is on L1\n")
			break
		}
		// 4. Assert the SourceProxyType is permitted, either "owner" or "spawn proxy"
		if tx.SourceProxyType != PROXY_OWNER && tx.SourceProxyType != PROXY_SPAWN {
			fmt.Printf("Ignoring tx: source proxy is not authorized to spawn\n")
			break
		}
		// 5. Assert the TargetShip isn't spawned yet (not in points map, in naive.hoon)
		var target Point
		err := db.DB.Get(&target, `select * from points where azimuth_number = ?`, tx.TargetShip)
		if err == nil {
			// Row found; point already exists
			fmt.Printf("Ignoring tx: target ship is already spawned\n")
			break
		} else if !errors.Is(err, sql.ErrNoRows) {
			// Unexpected error
			panic(err)
		}

		// 6. Create a new Point with sponsor=SourceShip and dominion=L2
		new_point := Point{
			Number:     tx.TargetShip,
			HasSponsor: true,
			Sponsor:    tx.SourceShip,
			Dominion:   2,
		}
		// 7. If SourceProxyType is "owner" and TargetAddress matches OwnerAddress, or likewise for
		// "spawn proxy", take ownership of it immediately (set p.OwnerAddress=TargetAddress)
		//    Otherwise:
		//    - p.OwnerAddress = parent.OwnerAddress;
		//    - p.TransferAddress = TargetAddress
		if tx.SourceProxyType == PROXY_OWNER && tx.TargetAddress == p.OwnerAddress ||
			tx.SourceProxyType == PROXY_SPAWN && tx.TargetAddress == p.SpawnAddress {
			new_point.OwnerAddress = tx.TargetAddress
		} else {
			new_point.OwnerAddress = p.OwnerAddress
			new_point.TransferAddress = tx.TargetAddress
		}
		diffs = append(diffs,
			AzimuthDiff{
				SourceEventLogID: tx.EthereumEventLogID,
				IntraLogIndex:    tx.IntraLogIndex,
				AzimuthNumber:    new_point.Number,
				Operation:        DIFF_SPAWNED,
			})
		// 8. Save the new point
		ret = append(ret,
			Query{`
				insert into points (azimuth_number, owner_address, transfer_address, has_sponsor, sponsor, dominion)
				            values (:azimuth_number, :owner_address, :transfer_address, :has_sponsor, :sponsor, :dominion)`,
				new_point,
			})
	case OP_CONFIGURE_KEYS:
		// 1. Assert SourceShip is on L2
		if p.Dominion != 2 {
			fmt.Printf("Ignoring tx: source ship is not on L2\n")
			break
		}
		// 2. Assert SourceProxyType is permitted, either "owner" or "management proxy"
		if tx.SourceProxyType != PROXY_OWNER && tx.SourceProxyType != PROXY_MANAGEMENT {
			fmt.Printf("Ignoring tx: source proxy is not authorized\n")
			break
		}

		// 3. if p.Flag (breach): increment p.Rift
		if tx.Flag {
			p.Rift += 1
			diffs = append(diffs,
				AzimuthDiff{
					SourceEventLogID: tx.EthereumEventLogID,
					IntraLogIndex:    tx.IntraLogIndex,
					AzimuthNumber:    p.Number,
					Operation:        DIFF_BREACHED,
				})
		}
		// 4. if p.Suite, p.Auth and p.Encrypt don't already match the new values, increment p.Life
		if p.CryptoSuiteVersion != tx.CryptoSuiteVersion ||
			!bytes.Equal(p.EncryptionKey, tx.EncryptionKey) ||
			!bytes.Equal(p.AuthKey, tx.AuthKey) {
			p.Life += 1

			diff_data := append(make([]byte, 4), append(tx.AuthKey, tx.EncryptionKey...)...)
			binary.BigEndian.PutUint32(diff_data[0:4], tx.CryptoSuiteVersion)
			diffs = append(diffs,
				AzimuthDiff{
					SourceEventLogID: tx.EthereumEventLogID,
					IntraLogIndex:    tx.IntraLogIndex,
					AzimuthNumber:    p.Number,
					Operation:        DIFF_RESET_KEYS,
					Data:             diff_data,
				})
		}
		// 5. set p.Suite = tx.Suite, p.Auth = tx.Auth, p.Encrypt = tx.Encrypt
		p.CryptoSuiteVersion = tx.CryptoSuiteVersion
		p.AuthKey = tx.AuthKey
		p.EncryptionKey = tx.EncryptionKey

		// 6. Save the point
		ret = append(ret,
			Query{`
				update points
				   set rift = :rift,
				       life = :life,
				       crypto_suite_version = :crypto_suite_version,
				       auth_key = :auth_key,
				       encryption_key = :encryption_key
				 where azimuth_number = :azimuth_number`,
				p,
			})
	case OP_ESCAPE:
		// 1. Assert SourceProxyType is permitted, either "owner" or "management"
		if tx.SourceProxyType != PROXY_OWNER && tx.SourceProxyType != PROXY_MANAGEMENT {
			fmt.Printf("Ignoring tx: source proxy is not authorized\n")
			break
		}
		// 2. Assert ranks match: TargetShip should be 1 rank higher than SourceShip
		if tx.TargetShip.Rank()+1 != tx.SourceShip.Rank() {
			fmt.Printf("Ignoring tx in event log %d: rank mismatch (%d/%d to %d/%d)\n",
				tx.EthereumEventLogID, tx.TargetShip, tx.TargetShip.Rank(), tx.SourceShip, tx.SourceShip.Rank(),
			)
			break
		}
		// 3. Apply escape request
		p.IsEscapeRequested = true
		p.EscapeRequestedTo = tx.TargetShip
		// 4. Save the point
		ret = append(ret,
			Query{`
				update points
				   set is_escape_requested = :is_escape_requested,
				       escape_requested_to = :escape_requested_to
				 where azimuth_number = :azimuth_number`,
				p,
			})
		diffs = append(diffs,
			AzimuthDiff{
				SourceEventLogID: tx.EthereumEventLogID,
				IntraLogIndex:    tx.IntraLogIndex,
				AzimuthNumber:    p.Number,
				Operation:        DIFF_ESCAPE_REQUESTED,
				Data:             azimuth_number_to_data(p.EscapeRequestedTo),
			})
	case OP_CANCEL_ESCAPE:
		// 1. Assert SourceProxyType is permitted, either "owner" or "management"
		if tx.SourceProxyType != PROXY_OWNER && tx.SourceProxyType != PROXY_MANAGEMENT {
			fmt.Printf("Ignoring tx: source proxy is not authorized\n")
			break
		}
		// 2. Apply escape cancellation
		p.IsEscapeRequested = false
		p.EscapeRequestedTo = 0
		// 3. Save the point
		ret = append(ret,
			Query{`
				update points
				   set is_escape_requested = :is_escape_requested,
				       escape_requested_to = :escape_requested_to
				 where azimuth_number = :azimuth_number`,
				p,
			})
		diffs = append(diffs,
			AzimuthDiff{
				SourceEventLogID: tx.EthereumEventLogID,
				IntraLogIndex:    tx.IntraLogIndex,
				AzimuthNumber:    p.Number,
				Operation:        DIFF_ESCAPE_CANCELED,
				Data:             azimuth_number_to_data(p.EscapeRequestedTo),
			})
	case OP_ADOPT:
		// 1. Assert SourceProxyType is permitted, either "owner" or "management"
		if tx.SourceProxyType != PROXY_OWNER && tx.SourceProxyType != PROXY_MANAGEMENT {
			fmt.Printf("Ignoring tx: source proxy is not authorized\n")
			break
		}

		target := get_point(tx.TargetShip)

		// 2. Assert tx.TargetShip has requested escape to tx.SourceShip
		if target.EscapeRequestedTo != tx.SourceShip {
			fmt.Printf("Ignoring tx in event log %d: target ship %d wasn't trying to escape to %d\n",
				tx.EthereumEventLogID, tx.TargetShip, tx.SourceShip,
			)
			break
		}
		// 3. Apply the adoption
		target.IsEscapeRequested = false
		target.EscapeRequestedTo = 0
		target.HasSponsor = true
		target.Sponsor = tx.SourceShip
		// 4. Save the point
		ret = append(ret,
			Query{`
				update points
				   set is_escape_requested = :is_escape_requested,
				       escape_requested_to = :escape_requested_to,
				       has_sponsor = :has_sponsor,
				       sponsor = :sponsor
				 where azimuth_number = :azimuth_number`,
				target,
			})
		diffs = append(diffs,
			AzimuthDiff{
				SourceEventLogID: tx.EthereumEventLogID,
				IntraLogIndex:    tx.IntraLogIndex,
				AzimuthNumber:    target.Number,
				Operation:        DIFF_ESCAPE_ACCEPTED,
				Data:             azimuth_number_to_data(target.EscapeRequestedTo),
			})
	case OP_REJECT:
		// 1. Assert SourceProxyType is permitted, either "owner" or "management"
		if tx.SourceProxyType != PROXY_OWNER && tx.SourceProxyType != PROXY_MANAGEMENT {
			fmt.Printf("Ignoring tx: source proxy is not authorized\n")
			break
		}

		target := get_point(tx.TargetShip)

		// 2. Assert tx.TargetShip has requested escape to tx.SourceShip
		if target.EscapeRequestedTo != tx.SourceShip {
			fmt.Printf("Ignoring tx in event log %d: target ship %d wasn't trying to escape to %d\n",
				tx.EthereumEventLogID, tx.TargetShip, tx.SourceShip,
			)
			break
		}
		// 3. Apply the rejection
		target.IsEscapeRequested = false
		target.EscapeRequestedTo = 0
		// 4. Save the point
		ret = append(ret,
			Query{`
				update points
				   set is_escape_requested = :is_escape_requested,
				       escape_requested_to = :escape_requested_to
				 where azimuth_number = :azimuth_number`,
				target,
			})
		diffs = append(diffs,
			AzimuthDiff{
				SourceEventLogID: tx.EthereumEventLogID,
				IntraLogIndex:    tx.IntraLogIndex,
				AzimuthNumber:    target.Number,
				Operation:        DIFF_ESCAPE_REJECTED,
				Data:             azimuth_number_to_data(target.EscapeRequestedTo),
			})
	case OP_DETACH: // Source ship (star) disavows target ship (planet)
		// 1. Assert SourceProxyType is permitted, either "owner" or "management"
		if tx.SourceProxyType != PROXY_OWNER && tx.SourceProxyType != PROXY_MANAGEMENT {
			fmt.Printf("Ignoring tx: source proxy is not authorized\n")
			break
		}

		target := get_point(tx.TargetShip)

		// 2. Assert source ship is currently the target's sponsor
		if tx.SourceShip != target.Sponsor {
			fmt.Printf("Ignoring tx in event log %d: source ship (%d) isn't target's (%d) sponsor\n",
				tx.EthereumEventLogID, tx.SourceShip, tx.TargetShip,
			)
			break
		}
		// 3. Apply the detachment
		target.HasSponsor = false
		target.Sponsor = 0

		// 4. Save the point
		ret = append(ret,
			Query{`
				update points
				   set has_sponsor = :has_sponsor,
				       sponsor = :sponsor
				 where azimuth_number = :azimuth_number`,
				target,
			})
		diffs = append(diffs,
			AzimuthDiff{
				SourceEventLogID: tx.EthereumEventLogID,
				IntraLogIndex:    tx.IntraLogIndex,
				AzimuthNumber:    target.Number,
				Operation:        DIFF_LOST_SPONSOR,
			})
	case OP_SET_MANAGEMENT_PROXY:
		// 1. Assert SourceShip is on L2
		if p.Dominion != 2 {
			fmt.Printf("Ignoring tx: source proxy is not on L2\n")
			break
		}
		// 2. Assert SourceProxyType is permitted, either "owner" or "management"
		if tx.SourceProxyType != PROXY_OWNER && tx.SourceProxyType != PROXY_MANAGEMENT {
			fmt.Printf("Ignoring tx: source proxy is not authorized\n")
			break
		}

		// 3. Update the proxy
		p.ManagementAddress = tx.TargetAddress
		// 4. Save the point
		ret = append(ret,
			Query{`
				update points
				   set management_address = :management_address
				 where azimuth_number = :azimuth_number`,
				p,
			})
		diffs = append(diffs,
			AzimuthDiff{
				SourceEventLogID: tx.EthereumEventLogID,
				IntraLogIndex:    tx.IntraLogIndex,
				AzimuthNumber:    p.Number,
				Operation:        DIFF_CHANGED_MANAGEMENT_PROXY,
				Data:             p.ManagementAddress[:],
			})
	case OP_SET_SPAWN_PROXY:
		// 1. Assert SourceShip is on L2 or "Spawn" dominion
		if p.Dominion != 2 && p.Dominion != 3 {
			fmt.Printf("Ignoring tx: source proxy is not on L2\n")
			break
		}
		// 2. Assert SourceProxyType is permitted, either "owner" or "spawn"
		if tx.SourceProxyType != PROXY_OWNER && tx.SourceProxyType != PROXY_SPAWN {
			fmt.Printf("Ignoring tx: source proxy is not authorized\n")
			break
		}
		// 3. Assert SourceShip is either a star or a galaxy (planets can't spawn)
		if tx.SourceShip.Rank() == PLANET {
			break
		}
		// 4. Update the proxy
		p.SpawnAddress = tx.TargetAddress
		// 5. Save the point
		ret = append(ret,
			Query{`
				update points
				   set spawn_address = :spawn_address
				 where azimuth_number = :azimuth_number`,
				p,
			})
		diffs = append(diffs,
			AzimuthDiff{
				SourceEventLogID: tx.EthereumEventLogID,
				IntraLogIndex:    tx.IntraLogIndex,
				AzimuthNumber:    p.Number,
				Operation:        DIFF_CHANGED_SPAWN_PROXY,
				Data:             p.SpawnAddress[:],
			})
	case OP_SET_TRANSFER_PROXY:
		// 1. Assert SourceShip is on L2
		if p.Dominion != 2 {
			fmt.Printf("Ignoring tx: source proxy is not on L2\n")
			break
		}
		// 2. Assert SourceProxyType is permitted, either "owner" or "transfer"
		if tx.SourceProxyType != PROXY_OWNER && tx.SourceProxyType != PROXY_TRANSFER {
			fmt.Printf("Ignoring tx: source proxy is not authorized\n")
			break
		}
		// 3. Update the proxy
		p.TransferAddress = tx.TargetAddress
		// 4. Save the point
		ret = append(ret,
			Query{`
				update points
				   set transfer_address = :transfer_address
				 where azimuth_number = :azimuth_number`,
				p,
			})
		diffs = append(diffs,
			AzimuthDiff{
				SourceEventLogID: tx.EthereumEventLogID,
				IntraLogIndex:    tx.IntraLogIndex,
				AzimuthNumber:    p.Number,
				Operation:        DIFF_CHANGED_TRANSFER_PROXY,
				Data:             p.TransferAddress[:],
			})
	default:
		panic(tx.Opcode)
	}
	return ret, diffs
}
