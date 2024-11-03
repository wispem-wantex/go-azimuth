package db

import (
	"github.com/ethereum/go-ethereum/common"
)

type Contract struct {
	ID                    uint64         `db:"rowid"`
	Address               common.Address `db:"address"`
	Name                  string         `db:"name"`
	StartBlockNum         uint64         `db:"start_block"`
	LatestBlockNumFetched uint64         `db:"latest_block_fetched"`
}

func (db *DB) GetContractByName(name string) Contract {
	var ret Contract
	query := `SELECT rowid, address, name, start_block, latest_block_fetched FROM contracts WHERE name like ?`
	err := db.DB.Get(&ret, query, name)
	if err != nil {
		panic(err)
	}
	return ret
}

// Update the contract to note that more blocks have been fetched, if applicable.  Won't go backward
func (db *DB) SetLatestContractBlockFetched(contract_id uint64, block_num uint64) {
	db.DB.MustExec(`UPDATE contracts SET latest_block_fetched = max(latest_block_fetched, ?) WHERE rowid = ?`, block_num, contract_id)
}
