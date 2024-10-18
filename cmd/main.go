package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/ethclient"

	"go-azimuth/pkg/crypto"
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
	case "store_key":
		if len(args) < 3 {
			panic("Gotta provide a ship and a key")
		}
		store_key(args[1], args[2])
	case "sign":
		if len(args) < 3 {
			panic("Gotta provide a ship or crubfile")
		}
		sign(args[1], args[2])
	case "verify":
		if len(args) < 4 {
			panic("Gotta provide: ship, signature, message")
		}
		verify(args[1], args[2], args[3])
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
		fmt.Printf("Not a valid ship name: %q\n", urbit_id)
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
		fmt.Printf("Not a valid ship name: %q\n", urbit_id)
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

func store_key(urbit_id string, key_hex string) {
	point, is_ok := phonemes.PhonemeToInt(urbit_id)
	if !is_ok {
		fmt.Printf("Not a valid ship name: %q\n", urbit_id)
		os.Exit(1)
	}
	db := get_db(DB_PATH)
	result, is_found := db.GetPoint(pkg_db.AzimuthNumber(point))
	if !is_found {
		fmt.Printf("Point not found!\n")
		os.Exit(2)
	}

	vein := crypto.UrbitVeinFromHex(key_hex)
	crub := vein.ToCrub()
	// Check the crub for validity
	if !bytes.Equal(crub.SignKeys.Pub[:], result.AuthKey) {
		fmt.Printf("Signing keys don't match:\n - expect: %x\n - actual: %x\n", result.AuthKey, crub.SignKeys.Pub)
	}
	if !bytes.Equal(crub.EncryptKeys.Pub[:], result.EncryptionKey) {
		fmt.Printf("Encrypt keys don't match:\n - expect: %x\n - actual: %x\n", result.EncryptionKey, crub.EncryptKeys.Pub)
	}

	// Save the crub
	file, err := os.Create(fmt.Sprintf("%s.crub", urbit_id))
	if err != nil {
		panic(err)
	}
	jsondata, err := json.Marshal(crub)
	if err != nil {
		panic(err)
	}
	_, err = file.Write(jsondata)
	if err != nil {
		panic(err)
	}
	err = file.Close()
	if err != nil {
		panic(err)
	}
}

func load_crubfile(urbit_id string) crypto.UrbitCrub {
	point, is_ok := phonemes.PhonemeToInt(urbit_id)
	if !is_ok {
		fmt.Printf("Not a valid ship name: %q\n", urbit_id)
		os.Exit(1)
	}
	db := get_db(DB_PATH)
	result, is_found := db.GetPoint(pkg_db.AzimuthNumber(point))
	if !is_found {
		fmt.Printf("Point not found!\n")
		os.Exit(2)
	}

	file, err := os.Open(fmt.Sprintf("%s.crub", urbit_id))
	if err != nil {
		panic(err)
	}
	jsondata, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}
	var crub crypto.UrbitCrub
	err = json.Unmarshal(jsondata, &crub)
	if err != nil {
		panic(err)
	}
	// Check the crub for validity
	if !bytes.Equal(crub.SignKeys.Pub[:], result.AuthKey) {
		fmt.Printf("Signing keys don't match:\n - expect: %x\n - actual: %x\n", result.AuthKey, crub.SignKeys.Pub)
	}
	if !bytes.Equal(crub.EncryptKeys.Pub[:], result.EncryptionKey) {
		fmt.Printf("Encrypt keys don't match:\n - expect: %x\n - actual: %x\n", result.EncryptionKey, crub.EncryptKeys.Pub)
	}
	return crub
}

func sign(urbit_id string, msg string) {
	crub := load_crubfile(urbit_id)
	signature := crub.Sign([]byte(msg))
	fmt.Printf("%x  %s\n", signature, msg)
}
func hex_to_bytes(s string) []byte {
	data, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return data
}
func verify(urbit_id string, signature string, msg string) {
	point, is_ok := phonemes.PhonemeToInt(urbit_id)
	if !is_ok {
		fmt.Printf("Not a valid ship name: %q\n", urbit_id)
		os.Exit(1)
	}
	db := get_db(DB_PATH)
	result, is_found := db.GetPoint(pkg_db.AzimuthNumber(point))
	if !is_found {
		fmt.Printf("Point not found!\n")
		os.Exit(2)
	}

	crub := crypto.UrbitCrub{}
	copy(crub.SignKeys.Pub[:], result.AuthKey)

	if crub.Verify(hex_to_bytes(signature), []byte(msg)) {
		fmt.Println("Signature is valid")
	} else {
		fmt.Println("Signature is not valid!")
	}
}
