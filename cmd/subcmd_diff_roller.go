package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	. "go-azimuth/pkg/db"
)

func dominionToString(code int) string {
	switch code {
	case 1:
		return "l1"
	case 2:
		return "l2"
	case 3:
		return "spawn"
	default:
		return fmt.Sprintf("l%d", code)
	}
}

func isZeroOrEmpty(b []byte) bool {
	if len(b) == 0 {
		return true
	}
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

func decodeRollerHexKey(s string) ([]byte, error) {
	if s == "" {
		return nil, nil
	}

	s = strings.TrimPrefix(s, "0x")

	if s == "" {
		// "0x" with no data: treat as no key
		return nil, nil
	}

	return hex.DecodeString(s)
}

func DiffDBPointWithRemote(dbp Point, rp RollerPoint) []string {
	diffs := []string{}

	// Dominion: db int vs API string
	dbDominion := dominionToString(dbp.Dominion)
	if !strings.EqualFold(dbDominion, rp.Dominion) {
		diffs = append(diffs, fmt.Sprintf("dominion: db=%s api=%s", dbDominion, rp.Dominion))
	}

	// Owner address
	if !strings.EqualFold(dbp.OwnerAddress.Hex(), rp.Ownership.Owner.Address) {
		diffs = append(diffs, fmt.Sprintf("owner address: db=%s api=%s",
			dbp.OwnerAddress.Hex(), rp.Ownership.Owner.Address))
	}

	// Owner nonce
	if dbp.OwnerNonce != uint32(rp.Ownership.Owner.Nonce) {
		diffs = append(diffs, fmt.Sprintf("owner nonce: db=%d api=%d",
			dbp.OwnerNonce, rp.Ownership.Owner.Nonce))
	}

	// Management proxy
	if !strings.EqualFold(dbp.ManagementAddress.Hex(), rp.Ownership.ManagementProxy.Address) {
		diffs = append(diffs, fmt.Sprintf("mgmt address: db=%s api=%s",
			dbp.ManagementAddress.Hex(), rp.Ownership.ManagementProxy.Address))
	}
	if dbp.ManagementNonce != uint32(rp.Ownership.ManagementProxy.Nonce) {
		diffs = append(diffs, fmt.Sprintf("mgmt nonce: db=%d api=%d",
			dbp.ManagementNonce, rp.Ownership.ManagementProxy.Nonce))
	}

	// Spawn proxy
	if !strings.EqualFold(dbp.SpawnAddress.Hex(), rp.Ownership.SpawnProxy.Address) {
		diffs = append(diffs, fmt.Sprintf("spawn address: db=%s api=%s",
			dbp.SpawnAddress.Hex(), rp.Ownership.SpawnProxy.Address))
	}
	if dbp.SpawnNonce != uint32(rp.Ownership.SpawnProxy.Nonce) {
		diffs = append(diffs, fmt.Sprintf("spawn nonce: db=%d api=%d",
			dbp.SpawnNonce, rp.Ownership.SpawnProxy.Nonce))
	}

	// Transfer proxy
	if !strings.EqualFold(dbp.TransferAddress.Hex(), rp.Ownership.TransferProxy.Address) {
		diffs = append(diffs, fmt.Sprintf("transfer address: db=%s api=%s",
			dbp.TransferAddress.Hex(), rp.Ownership.TransferProxy.Address))
	}
	if dbp.TransferNonce != uint32(rp.Ownership.TransferProxy.Nonce) {
		diffs = append(diffs, fmt.Sprintf("transfer nonce: db=%d api=%d",
			dbp.TransferNonce, rp.Ownership.TransferProxy.Nonce))
	}

	// Rift
	if rift, err := strconv.ParseUint(rp.Network.Rift, 10, 32); err == nil {
		if dbp.Rift != uint32(rift) {
			diffs = append(diffs, fmt.Sprintf("rift: db=%d api=%d", dbp.Rift, rift))
		}
	} else {
		diffs = append(diffs, fmt.Sprintf("rift: invalid api value %q", rp.Network.Rift))
	}

	// Keys
	if crypt, err := decodeRollerHexKey(rp.Network.Keys.Crypt); err == nil {
		if isZeroOrEmpty(dbp.EncryptionKey) && isZeroOrEmpty(crypt) {
			// both effectively "no key" -> OK
		} else if !bytes.Equal(dbp.EncryptionKey, crypt) {
			diffs = append(diffs,
				fmt.Sprintf("encryption key mismatch: db: %v roller: %v", dbp.EncryptionKey, crypt))
		}
	} else {
		fmt.Printf("db: %v roller: %v rp: %v\n", dbp.EncryptionKey, nil, rp.Network.Keys.Crypt)
		diffs = append(diffs, "invalid crypt key in api")
	}

	if auth, err := decodeRollerHexKey(rp.Network.Keys.Auth); err == nil {
		if isZeroOrEmpty(dbp.AuthKey) && isZeroOrEmpty(auth) {
			// both effectively "no key" -> OK
		} else if !bytes.Equal(dbp.AuthKey, auth) {
			diffs = append(diffs,
				fmt.Sprintf("auth key mismatch: db: %v roller: %v", dbp.AuthKey, auth))
		}
	} else {
		fmt.Printf("db: %v roller: %v rp: %v\n", dbp.AuthKey, nil, rp.Network.Keys.Auth)
		diffs = append(diffs, "invalid auth key in api")
	}

	if life, err := strconv.ParseUint(rp.Network.Keys.Life, 10, 32); err == nil {
		if dbp.Life != uint32(life) {
			diffs = append(diffs, fmt.Sprintf("life: db=%d api=%d", dbp.Life, life))
		}
	} else {
		diffs = append(diffs, fmt.Sprintf("life: invalid api value %q", rp.Network.Keys.Life))
	}

	if suite, err := strconv.ParseUint(rp.Network.Keys.Suite, 10, 32); err == nil {
		if dbp.CryptoSuiteVersion != uint32(suite) {
			diffs = append(diffs, fmt.Sprintf("crypto suite: db=%d api=%d",
				dbp.CryptoSuiteVersion, suite))
		}
	} else {
		diffs = append(diffs, fmt.Sprintf("suite: invalid api value %q", rp.Network.Keys.Suite))
	}

	// Sponsor
	if dbp.HasSponsor != rp.Network.Sponsor.Has {
		diffs = append(diffs, fmt.Sprintf("hasSponsor: db=%v api=%v",
			dbp.HasSponsor, rp.Network.Sponsor.Has))
	}
	if rp.Network.Sponsor.Has {
		if dbp.Sponsor != AzimuthNumber(rp.Network.Sponsor.Who) {
			diffs = append(diffs, fmt.Sprintf("sponsor: db=%d api=%d",
				dbp.Sponsor, rp.Network.Sponsor.Who))
		}
	}

	// Escape requests
	escapeRequested := rp.Network.Escape != nil
	if dbp.IsEscapeRequested != escapeRequested {
		diffs = append(diffs, fmt.Sprintf("isEscapeRequested: db=%v api=%v",
			dbp.IsEscapeRequested, escapeRequested))
	}
	if escapeRequested && dbp.EscapeRequestedTo != *rp.Network.Escape {
		diffs = append(diffs, fmt.Sprintf("escapeRequestedTo: db=%d api=%d",
			dbp.EscapeRequestedTo, *rp.Network.Escape))
	}
	return diffs
}

type RollerPoint struct {
	Dominion  string `json:"dominion"`
	Ownership struct {
		Owner struct {
			Address string `json:"address"`
			Nonce   int    `json:"nonce"`
		} `json:"owner"`
		ManagementProxy struct {
			Address string `json:"address"`
			Nonce   int    `json:"nonce"`
		} `json:"managementProxy"`
		SpawnProxy struct {
			Address string `json:"address"`
			Nonce   int    `json:"nonce"`
		} `json:"spawnProxy"`
		TransferProxy struct {
			Address string `json:"address"`
			Nonce   int    `json:"nonce"`
		} `json:"transferProxy"`
	} `json:"ownership"`
	Network struct {
		Escape *AzimuthNumber `json:"escape,omitempty"`
		Keys   struct {
			Life  string `json:"life"`
			Suite string `json:"suite"`
			Auth  string `json:"auth"`
			Crypt string `json:"crypt"`
		} `json:"keys"`
		Sponsor struct {
			Has  bool   `json:"has"`
			Who  int    `json:"who"`
			Patp string `json:"patp"`
		} `json:"sponsor"`
		Sein struct {
			Has  bool   `json:"has"`
			Who  string `json:"who"`
			Patp string `json:"patp"`
		} `json:"sein"`
		Rift string `json:"rift"`
	} `json:"network"`
}

func CheckPointsAgainstRoller(db DB, url string) error {
	points, ok := db.GetPoints()
	if !ok || len(points) == 0 {
		fmt.Println("no points in DB")
		return nil
	}

	for _, p := range points {
		reqBody, err := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      "7",
			"method":  "getPoint",
			"params":  map[string]interface{}{"ship": int(p.Number)},
		})
		if err != nil {
			return fmt.Errorf("marshal request for point %d: %w", p.Number, err)
		}

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			fmt.Println("roller getpoint error")
			return fmt.Errorf("roller GetPoint(%d): %w", p.Number, err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read response for point %d: %w", p.Number, err)
		}

		var jsonRPCResp struct {
			Result RollerPoint `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &jsonRPCResp); err != nil {
			return fmt.Errorf("unmarshal jsonrpc response for point %d: %w", p.Number, err)
		}

		if jsonRPCResp.Error != nil {
			return fmt.Errorf("jsonrpc error for point %d: %s", p.Number, jsonRPCResp.Error.Message)
		}

		diffs := DiffDBPointWithRemote(p, jsonRPCResp.Result)
		if len(diffs) > 0 {
			fmt.Printf("point %d mismatches:\n", p.Number)
			for _, d := range diffs {
				fmt.Printf("  - %s\n", d)
			}
		}
	}

	return nil
}
