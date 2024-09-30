package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/ethclient"

	pkg_db "go-azimuth/pkg/db"
	"go-azimuth/pkg/phonemes"
	"go-azimuth/pkg/scraper"
)

const (
	RPC_URL = "https://cloudflare-eth.com/v1/mainnet"
)

var DB_PATH = ""

func get_db(path string) pkg_db.DB {
	db, err := pkg_db.DBCreate(path)
	if errors.Is(err, pkg_db.ErrTargetExists) {
		db, err = pkg_db.DBConnect(path)
		if err != nil {
			panic(err)
		}
	}
	return db
}

func main() {
	flag.StringVar(&DB_PATH, "db", "sample_data/azimuth.db", "")

	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		fmt.Printf("subcommand needed\n")
		os.Exit(1)
	}

	switch args[0] {
	case "get_logs":
		flags := flag.NewFlagSet("", flag.ExitOnError)
		with_apply := flags.Bool("apply-logs", false, "whether to apply the logs immediately as they're fetched (default no)")
		if err := flags.Parse(args[1:]); err != nil {
			panic(err)
		}
		catch_up_logs(*with_apply)
	case "play_logs":
		play_logs()
	case "query":
		query(args[1])
	default:
		fmt.Printf("invalid subcommand: %q\n", args[0])
		os.Exit(1)
	}
}

func catch_up_logs(with_apply bool) {
	db := get_db(DB_PATH)
	client, err := ethclient.Dial(RPC_URL)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}
	defer client.Close()

	scraper.CatchUpAzimuthLogs(client, db, with_apply)
}

func play_logs() {
	db := get_db(DB_PATH)
	db.PlayAzimuthLogs()
}

func query(urbit_id string) {
	_point, is_ok := phonemes.PhonemeToInt(urbit_id)
	if !is_ok {
		fmt.Printf("Not a valid phoneme: %q\n", urbit_id)
		os.Exit(1)
	}
	point := phonemes.Unscramble(uint32(_point))
	fmt.Printf("Querying point %d\n", point)

	db := get_db(DB_PATH)
	result, is_found := db.GetPoint(pkg_db.AzimuthNumber(point))
	if !is_found {
		fmt.Printf("Point not found!\n")
		os.Exit(2)
	} else {
		fmt.Printf("%#v\n", result)
	}
}
