package main

import (
	"encoding/json"
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

var ETHEREUM_RPC_URL = ""

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

func require_eth_rpc_url() {
	if ETHEREUM_RPC_URL == "" {
		fmt.Printf(
			"This command requires an ethereum RPC url.  You can provide one using the environment variable " +
				"\"ETHEREUM_RPC_URL\", or with the '--eth-url' flag.\n\n" +
				"You can get one for free at: https://www.infura.io\n\n" +
				"Once you get an API key, do\n\n" +
				"    export ETHEREUM_RPC_URL=https://mainnet.infura.io/v3/<YOUR_API_KEY>\n\n" +
				"and then try running this command again.\n")
		os.Exit(1)
	}
}

func main() {
	if val, is_ok := os.LookupEnv("ETHEREUM_RPC_URL"); is_ok {
		ETHEREUM_RPC_URL = val
	}

	flag.StringVar(&DB_PATH, "db", "azimuth.db", "database file")
	flag.StringVar(&ETHEREUM_RPC_URL, "eth-url", ETHEREUM_RPC_URL,
		"Ethereum node RPC URL (defaults to environment variable ETHEREUM_RPC_URL)")

	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		fmt.Printf("subcommand needed\n")
		os.Exit(1)
	}

	switch args[0] {
	case "get_logs_azimuth":
		catch_up_logs()
	case "play_logs_azimuth":
		play_logs()
	case "query":
		query(args[1])
	case "show_logs":
		show_logs(args[1])
	case "get_logs_naive":
		catch_up_logs_naive()
	case "play_logs_naive":
		play_naive_logs()
	case "checkpoint":
		if len(args) < 2 {
			panic("Gotta provide a path to checkpoint into")
		}
		checkpoint(args[1])
	default:
		fmt.Printf("invalid subcommand: %q\n", args[0])
		os.Exit(1)
	}
}

func catch_up_logs() {
	require_eth_rpc_url()

	db := get_db(DB_PATH)
	client, err := ethclient.Dial(ETHEREUM_RPC_URL)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}
	defer client.Close()

	scraper.CatchUpAzimuthLogs(client, db)
}

func catch_up_logs_naive() {
	require_eth_rpc_url()

	db := get_db(DB_PATH)
	client, err := ethclient.Dial(ETHEREUM_RPC_URL)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}
	defer client.Close()

	scraper.CatchUpNaiveLogs(client, db, false)
}

func play_logs() {
	db := get_db(DB_PATH)
	db.PlayAzimuthLogs()
}
func play_naive_logs() {
	db := get_db(DB_PATH)
	db.PlayNaiveLogs()
}

func query(urbit_id string) {
	point, is_ok := phonemes.PhonemeToInt(urbit_id)
	if !is_ok {
		fmt.Printf("Not a valid phoneme: %q\n", urbit_id)
		os.Exit(1)
	}
	println(fmt.Sprintf("Querying point %d\n", point))

	db := get_db(DB_PATH)
	result, is_found := db.GetPoint(pkg_db.AzimuthNumber(point))
	if !is_found {
		fmt.Printf("Point not found!\n")
		os.Exit(2)
	} else {
		data, err := json.Marshal(result)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(data))
	}
}

func show_logs(urbit_id string) {
	point, is_ok := phonemes.PhonemeToInt(urbit_id)
	if !is_ok {
		fmt.Printf("Not a valid phoneme: %q\n", urbit_id)
		os.Exit(1)
	}
	db := get_db(DB_PATH)
	result, is_found := db.GetEventsForPoint(pkg_db.AzimuthNumber(point))
	if !is_found {
		fmt.Printf("Point not found!\n")
		os.Exit(2)
	}

	// Header
	fmt.Printf("%-7s  %-7s  %-64s  %-3s  %-24s  %s\n", "ID", "Layer", "Tx Hash", "Idx", "Operation", "Data")
	fmt.Printf("-------  -------  ----------------------------------------------------------------  ---  ------------------------  ----\n")
	for _, h := range result {
		fmt.Printf("%-7d  %-7s  %-64s  %-3d  %-24s  %s\n",
			h.ID, h.ContractName, h.TxHash, h.IntraLogIndex, h.OperationName, h.HexData)
	}
}

func checkpoint(path string) {
	db := get_db(DB_PATH)
	fmt.Println("Vaccuuming")
	db.DB.MustExec(`vacuum into ?`, path)
	fmt.Println("Vaccuumed")
}
