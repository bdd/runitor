// Copyright 2020 - 2022, Berk D. Demir and the runitor contributors
// SPDX-License-Identifier: 0BSD
package internal

import (
	"crypto/rand"
	"encoding/hex"
)

const uuid4RandBytes = 128 / 8

func NewUUID4() (string, error) {
	rnd := make([]byte, uuid4RandBytes)
	_, err := rand.Read(rnd)
	if err != nil {
		return "", err
	}

	rnd[6] = (rnd[6] & 0x0f) | 0x40 // version 4
	rnd[8] = (rnd[8] & 0x3f) | 0x80 // variant 0b10

	var str [2*uuid4RandBytes + 4]byte
	//       \______________/ \_/
	//              |          |_> 4 '-' separators
	//              |____________> 2 hex digits per byte
	hex.Encode(str[:], rnd[:4])
	str[8] = '-'
	hex.Encode(str[9:13], rnd[4:6])
	str[13] = '-'
	hex.Encode(str[14:18], rnd[6:8])
	str[18] = '-'
	hex.Encode(str[19:23], rnd[8:10])
	str[23] = '-'
	hex.Encode(str[24:], rnd[10:])

	return string(str[:]), nil
}
