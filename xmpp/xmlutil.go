// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package xmpp

import (
	"bytes"
	"encoding/xml"
)

func XMLEscape(text string) string {
	buf := new(bytes.Buffer)
	xml.EscapeText(buf, []byte(text))
	return buf.String()
}
