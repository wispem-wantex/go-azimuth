package db_test

import (
	"testing"
	"fmt"
	"math/rand"

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

func TestCreateDB(t *testing.T) {
	_, err := DBCreate(fmt.Sprintf("../../sample_data/random-%d.db", rand.Uint32()))
	assert.NoError(t, err)
}
