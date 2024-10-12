package db

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
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

	// Naive Events
	BATCH = get_hash("Batch()")
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

	EVENT_NAMES[BATCH] = "Batch"
}

var L2_DEPOSIT_ADDRESS = common.HexToAddress("1111111111111111111111111111111111111111")

type Query struct {
	SQL        string
	BindValues interface{}
}

type EthereumEventLog struct {
	ID          uint64      `db:"rowid"`
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

func (db *DB) SaveEvent(e *EthereumEventLog) {
	fmt.Printf("%#v\n", e)
	result, err := db.DB.NamedExec(`
		insert into ethereum_events (
			            block_number, block_hash, tx_hash, log_index, contract_address, topic0, topic1, topic2, data, is_processed
			        ) values (
			            :block_number, :block_hash, :tx_hash, :log_index, :contract_address, :topic0, :topic1, :topic2, :data,
			            :is_processed
			        )
	`, e)
	if err != nil {
		panic(err)
	}

	// Update the event's ID
	new_id, err := result.LastInsertId()
	if err != nil {
		panic(err)
	}
	e.ID = uint64(new_id)
}

// Either create a new event, or add in the Naive Batch data after the fact
func (db *DB) SmuggleNaiveBatchDataIntoEvent(e EthereumEventLog) {
	rslt, err := db.DB.NamedExec(`update ethereum_events set data=:data where block_number=:block_number and log_index=:log_index`, e)
	if err != nil {
		panic(err)
	}
	// Ensure that a row was updated
	rows_affected, err := rslt.RowsAffected()
	if err != nil {
		panic(err)
	} else if rows_affected != 1 {
		panic(e)
	}
}

// Play Azimuth logs until the first Naive tx.
//
// WTF(naive-azimuth-interlacing): Note that L1 and L2 txs actually do have to be
// processed in-order; the L1 does not "happen before" the L2, as I had previously believed.
func (db *DB) PlayAzimuthLogs() {
	var events []EthereumEventLog
	for {
		// Batches of 500.  Go until the Naive contract starts
		err := db.DB.Select(&events, `
		    select rowid, block_number, block_hash, tx_hash, log_index, contract_address, topic0, topic1,
		            topic2, data, is_processed from ethereum_events
		     where contract_address = X'223c067f8cf28ae173ee5cafea60ca44c335fecb' and is_processed = 0
		       and block_number < (select start_block from contracts where name like 'Naive')
		  order by block_number, log_index asc
		     limit 500
		`)
		if err != nil {
			panic(err)
		} else if len(events) == 0 {
			// No unprocessed logs left; we're finished
			break
		}
		db.ApplyEventEffects(events)
	}
}

func (db *DB) ApplyEventEffects(events []EthereumEventLog) {
	tx, err := db.DB.Begin()
	if err != nil {
		panic(err)
	}

	for _, e := range events {
		effects, diffs := e.Effects()

		// Apply the query
		if effects.SQL != "" {
			_, err = db.DB.NamedExec(effects.SQL, effects.BindValues)
			if err != nil {
				fmt.Printf("%q; %#v\n", effects.SQL, effects.BindValues)
				if err := tx.Rollback(); err != nil {
					panic(err)
				}
				panic(err)
			}
		}

		// Save the diffs
		for _, d := range diffs {
			db.SaveDiff(d)
		}

		// Mark the event as processed
		_, err = db.DB.NamedExec(`
			update ethereum_events
			   set is_processed=1
			 where block_number = :block_number and log_index = :log_index`,
			e)
		if err != nil {
			if err := tx.Rollback(); err != nil {
				panic(err)
			}
			panic(err)
		}
	}
	if err = tx.Commit(); err != nil {
		panic(err)
	}
}

func topic_to_uint32(h common.Hash) uint32 {
	// Topics are 32 bytes; uint32s are 4 bytes; so remove the first 28
	return binary.BigEndian.Uint32([]byte(h[28:]))
}
func topic_to_azimuth_number(h common.Hash) AzimuthNumber {
	return AzimuthNumber(topic_to_uint32(h))
}
func topic_to_eth_address(h common.Hash) common.Address {
	// Topics are 32 bytes; addresses are 20; so remove the first 12
	return common.BytesToAddress(h[:])
}
func azimuth_number_to_data(u AzimuthNumber) []byte {
	ret := make([]byte, 4)
	binary.BigEndian.PutUint32(ret, uint32(u))
	return ret
}

func (e EthereumEventLog) Effects() (Query, []AzimuthDiff) {
	if topic_to_azimuth_number(e.Topic1) == 1696251928 {
		fmt.Printf("Tiller Tolbus L1 event: %#v\n", e)
	}

	switch e.Topic0 {
	case SPAWNED:
		p := Point{
			Number: topic_to_azimuth_number(e.Topic2),
		}
		return Query{`insert into points (azimuth_number) values (:azimuth_number)`, p},
			[]AzimuthDiff{{SourceEventLogID: e.ID, IntraLogIndex: 0, AzimuthNumber: p.Number, Operation: DIFF_SPAWNED}}

	case ACTIVATED:
		p := Point{
			Number:     topic_to_azimuth_number(e.Topic1),
			IsActive:   true,
			HasSponsor: true,
		}
		if p.Number < 0x10000 {
			// A star's original sponsor is a galaxy
			p.Sponsor = p.Number % 0x100
		} else {
			// A planet's original sponsor is a star
			p.Sponsor = p.Number % 0x10000
		}
		return Query{`
			insert into points (azimuth_number, is_active, has_sponsor, sponsor)
			            values (:azimuth_number, :is_active, :has_sponsor, :sponsor)
			on conflict do update
			        set is_active=:is_active,
			            has_sponsor=:has_sponsor,
			            sponsor=:sponsor`,
				p,
			},
			[]AzimuthDiff{{SourceEventLogID: e.ID, IntraLogIndex: 0, AzimuthNumber: p.Number, Operation: DIFF_ACTIVATED}}

	case OWNER_CHANGED:
		p := Point{
			Number:       topic_to_azimuth_number(e.Topic1),
			OwnerAddress: topic_to_eth_address(e.Topic2),
		}
		if p.OwnerAddress == L2_DEPOSIT_ADDRESS {
			// Deposited to L2
			p.Dominion = 2
			return Query{`
				insert into points (azimuth_number, dominion)
				            values (:azimuth_number, :dominion)
				on conflict do update
				        set dominion=:dominion`,
					p,
				},
				[]AzimuthDiff{{
					SourceEventLogID: e.ID,
					IntraLogIndex:    0,
					AzimuthNumber:    p.Number,
					Operation:        DIFF_NEW_DOMINION,
					Data:             []byte{2},
				}}
		} else {
			return Query{`
				insert into points (azimuth_number, owner_address)
							values (:azimuth_number, :owner_address)
				on conflict do update
				        set owner_address=:owner_address
			          where dominion != 2`,
					p,
				},
				[]AzimuthDiff{{
					SourceEventLogID: e.ID,
					IntraLogIndex:    0,
					AzimuthNumber:    p.Number,
					Operation:        DIFF_CHANGED_OWNER,
					Data:             p.OwnerAddress[:],
				}}
		}
	case CHANGED_SPAWN_PROXY:
		p := Point{
			Number:       topic_to_azimuth_number(e.Topic1),
			SpawnAddress: topic_to_eth_address(e.Topic2),
		}
		if p.Number <= 0xffff && p.SpawnAddress == L2_DEPOSIT_ADDRESS {
			// Setting spawn proxy to the L2 deposit address represents migrating to the "Spawn"
			// dominion.  It's not a real / valid change of spawn proxy address.
			// In the "spawn" dominion, the ship (star or galaxy) is on L1, but it spawns on L2.
			//
			// Ships on L2 can't update their spawn proxy on L1, so it's safe to assume that the
			// ship is currently L1.
			p.Dominion = 3 // "spawn" dominion; ship is on L1, but spawns on L2
			return Query{`
				insert into points (azimuth_number, dominion)
				            values (:azimuth_number, :dominion)
				on conflict do update
				        set dominion=:dominion`,
					p,
				},
				[]AzimuthDiff{{
					SourceEventLogID: e.ID,
					IntraLogIndex:    0,
					AzimuthNumber:    p.Number,
					Operation:        DIFF_NEW_DOMINION,
					Data:             []byte{3},
				}}
		} else {
			// Actual change of spawn-proxy address
			return Query{`
				insert into points (azimuth_number, spawn_address)
				            values (:azimuth_number, :spawn_address)
				on conflict do update
				        set spawn_address=:spawn_address`,
					p,
				},
				[]AzimuthDiff{{
					SourceEventLogID: e.ID,
					IntraLogIndex:    0,
					AzimuthNumber:    p.Number,
					Operation:        DIFF_CHANGED_SPAWN_PROXY,
					Data:             p.SpawnAddress[:],
				}}
		}
	case CHANGED_TRANSFER_PROXY:
		p := Point{
			Number:          topic_to_azimuth_number(e.Topic1),
			TransferAddress: topic_to_eth_address(e.Topic2),
		}
		return Query{`
			insert into points (azimuth_number, transfer_address)
			            values (:azimuth_number, :transfer_address)
			on conflict do update
			        set transfer_address=:transfer_address
			      where dominion != 2`,
				p,
			},
			[]AzimuthDiff{{
				SourceEventLogID: e.ID,
				IntraLogIndex:    0,
				AzimuthNumber:    p.Number,
				Operation:        DIFF_CHANGED_TRANSFER_PROXY,
				Data:             p.TransferAddress[:],
			}}
	case CHANGED_MANAGEMENT_PROXY:
		p := Point{
			Number:            topic_to_azimuth_number(e.Topic1),
			ManagementAddress: topic_to_eth_address(e.Topic2),
		}
		return Query{`
			insert into points (azimuth_number, management_address)
			            values (:azimuth_number, :management_address)
			on conflict do update
			        set management_address=:management_address
			      where dominion != 2`,
				p,
			},
			[]AzimuthDiff{{
				SourceEventLogID: e.ID,
				IntraLogIndex:    0,
				AzimuthNumber:    p.Number,
				Operation:        DIFF_CHANGED_MANAGEMENT_PROXY,
				Data:             p.ManagementAddress[:],
			}}
	case CHANGED_VOTING_PROXY:
		p := Point{
			Number:        topic_to_azimuth_number(e.Topic1),
			VotingAddress: topic_to_eth_address(e.Topic2),
		}
		return Query{`
			insert into points (azimuth_number, voting_address)
			            values (:azimuth_number, :voting_address)
			on conflict do update
			        set voting_address=:voting_address
			      where dominion != 2`,
				p,
			},
			[]AzimuthDiff{{
				SourceEventLogID: e.ID,
				IntraLogIndex:    0,
				AzimuthNumber:    p.Number,
				Operation:        DIFF_CHANGED_VOTING_PROXY,
				Data:             p.VotingAddress[:],
			}}
	case ESCAPE_REQUESTED:
		p := Point{
			Number:            topic_to_azimuth_number(e.Topic1),
			IsEscapeRequested: true,
			EscapeRequestedTo: topic_to_azimuth_number(e.Topic2),
		}
		return Query{`
			insert into points (azimuth_number, is_escape_requested, escape_requested_to)
			            values (:azimuth_number, 1, :escape_requested_to)
			on conflict do update
			        set is_escape_requested=1,
			            escape_requested_to=:escape_requested_to
			      where dominion != 2`,
				p,
			},
			[]AzimuthDiff{{
				SourceEventLogID: e.ID,
				IntraLogIndex:    0,
				AzimuthNumber:    p.Number,
				Operation:        DIFF_ESCAPE_REQUESTED,
				Data:             azimuth_number_to_data(p.EscapeRequestedTo),
			}}
	case ESCAPE_CANCELED:
		p := Point{
			Number:            topic_to_azimuth_number(e.Topic1),
			IsEscapeRequested: false,
			EscapeRequestedTo: AzimuthNumber(0),
		}
		return Query{`
			insert into points (azimuth_number, is_escape_requested, escape_requested_to)
			            values (:azimuth_number, 0, 0)
			on conflict do update
			        set is_escape_requested=0, escape_requested_to=0
			      where dominion != 2`,
				p,
			},
			[]AzimuthDiff{{
				SourceEventLogID: e.ID,
				IntraLogIndex:    0,
				AzimuthNumber:    p.Number,
				Operation:        DIFF_ESCAPE_CANCELED,
				Data:             azimuth_number_to_data(p.EscapeRequestedTo),
			}}
	case ESCAPE_ACCEPTED:
		// TODO: ignore event if new sponsor is L2 (not sure how this could happen, but it's in naive.hoon)
		p := Point{
			Number:            topic_to_azimuth_number(e.Topic1),
			IsEscapeRequested: false,
			EscapeRequestedTo: AzimuthNumber(0),
			HasSponsor:        true,
			Sponsor:           topic_to_azimuth_number(e.Topic2),
		}
		return Query{`
			insert into points (azimuth_number, is_escape_requested, escape_requested_to, has_sponsor, sponsor)
			            values (:azimuth_number, :is_escape_requested, :escape_requested_to, :has_sponsor, :sponsor)
		    on conflict do update
		            set is_escape_requested=:is_escape_requested,
		                escape_requested_to=:escape_requested_to,
		                has_sponsor=:has_sponsor,
		                sponsor=:sponsor`,
				p,
			},
			[]AzimuthDiff{{
				SourceEventLogID: e.ID,
				IntraLogIndex:    0,
				AzimuthNumber:    p.Number,
				Operation:        DIFF_ESCAPE_ACCEPTED,
				Data:             azimuth_number_to_data(p.EscapeRequestedTo),
			}}
	case LOST_SPONSOR:
		// TODO: ignore event if lost sponsor (e.Topic2) isn't this point's sponsor
		// TODO: ignore event if new sponsor is L2 (not sure how this could happen, but it's in naive.hoon)
		p := Point{
			Number:     topic_to_azimuth_number(e.Topic1),
			HasSponsor: false,
		}
		return Query{`
			insert into points (azimuth_number, has_sponsor)
			            values (:azimuth_number, :has_sponsor)
			on conflict do update
			        set has_sponsor=:has_sponsor`,
				p,
			},
			[]AzimuthDiff{{
				SourceEventLogID: e.ID,
				IntraLogIndex:    0,
				AzimuthNumber:    p.Number,
				Operation:        DIFF_LOST_SPONSOR,
			}}
	case BROKE_CONTINUITY:
		p := Point{
			Number: topic_to_azimuth_number(e.Topic1),
			Rift:   topic_to_uint32(e.Topic2),
		}
		return Query{`
			insert into points (azimuth_number, rift)
			            values (:azimuth_number, :rift)
			on conflict do update
			        set rift=:rift
			      where dominion != 2`,
				p,
			},
			[]AzimuthDiff{{
				SourceEventLogID: e.ID,
				IntraLogIndex:    0,
				AzimuthNumber:    p.Number,
				Operation:        DIFF_BREACHED,
			}}
	case CHANGED_KEYS:
		if len(e.Data) != 32*4 { // Four 32-byte EVM words
			panic(len(e.Data))
		}
		p := Point{
			Number:             topic_to_azimuth_number(e.Topic1),
			EncryptionKey:      e.Data[:32],
			AuthKey:            e.Data[32 : 32*2],
			CryptoSuiteVersion: binary.BigEndian.Uint32([]byte(e.Data[32*3-4 : 32*3])),
			Life:               binary.BigEndian.Uint32([]byte(e.Data[32*4-4 : 32*4])),
		}
		return Query{`
			insert into points (azimuth_number, encryption_key, auth_key, crypto_suite_version, life)
			            values (:azimuth_number, :encryption_key, :auth_key, :crypto_suite_version, :life)
			on conflict do update
			        set encryption_key=:encryption_key,
			            auth_key=:auth_key,
			            crypto_suite_version=:crypto_suite_version,
			            life=:life
			      where dominion != 2`,
				p,
			},
			[]AzimuthDiff{{
				SourceEventLogID: e.ID,
				IntraLogIndex:    0,
				AzimuthNumber:    p.Number,
				Operation:        DIFF_RESET_KEYS,
				Data:             e.Data,
			}}
	case CHANGED_DNS:
		return Query{}, []AzimuthDiff{} // TODO
	default:
		panic(e.Topic0)
	}
}
