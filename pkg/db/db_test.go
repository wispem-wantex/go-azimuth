package db_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	. "go-azimuth/pkg/db"
)

func get_db(name string) DB {
	db, err := DBConnect("../../sample_data/" + name)
	if err != nil {
		panic(err)
	}
	return db
}

func TestCreateAndConnectToDB(t *testing.T) {
	i := rand.Uint32()
	_, err := DBCreate(fmt.Sprintf("../../sample_data/random-%d.db", i))
	assert.NoError(t, err)

	_, err = DBConnect(fmt.Sprintf("../../sample_data/random-%d.db", i))
	assert.NoError(t, err)
}
