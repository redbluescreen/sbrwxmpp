// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package xmpp

import (
	"crypto/rand"
	"math/big"
)

const randomCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func RandomStringSecure(length int) string {
	out := make([]byte, length)
	for i := 0; i < length; i++ {
		bigIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(randomCharset))))
		index := int(bigIndex.Int64())
		out[i] = randomCharset[index]
	}
	return string(out)
}
