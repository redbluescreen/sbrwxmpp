// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package xmlstream

import (
	"strings"
	"testing"
)

var testDoc1 string = `<?xml version="1.0" encoding="UTF-8" ?>
<root>
	<child id="1">
		<info>
			<id>1</id>
		</info>
		<info>
			<id>2</id>
		</info>
		<info>
			<id>3</id>
		</info>
	</child>
	<child id="2">
		Hello, world!
	</child>
	<child id="3">
	</child>
</root>
`

func TestIterateStreamChildren(t *testing.T) {
	reader := strings.NewReader(testDoc1)
	el, err := NewStream(reader)
	if err != nil {
		panic(err)
	}
	i := 0
	for {
		_, err := el.NextChild()
		if err == NoMoreChildrenError && i == 3 {
			break
		}
		if err != nil {
			panic(err)
		}
		i++
	}
	_, err = el.NextChild()
	if err != NoMoreChildrenError {
		panic(err)
	}
}

func TestGetAttr(t *testing.T) {
	reader := strings.NewReader(testDoc1)
	el, err := NewStream(reader)
	if err != nil {
		panic(err)
	}
	ch, err := el.NextChild()
	if err != nil {
		panic(err)
	}
	if ch.GetAttr("id") != "1" {
		t.Fatal("attr id not 1")
	}
	if ch.GetAttr("abc") != "" {
		t.Fatal("attr abc not empty")
	}
}

func TestGetChild(t *testing.T) {
	reader := strings.NewReader(testDoc1)
	el, err := NewStream(reader)
	if err != nil {
		panic(err)
	}
	ch, err := el.NextChild()
	if err != nil {
		panic(err)
	}
	_, ok := ch.GetChild("id")
	if ok {
		t.Fatal("GetChild(id) is ok")
	}
	info, ok := ch.GetChild("info")
	if !ok {
		t.Fatal("GetChild(info) not ok")
	}
	_, ok = info.GetChild("id")
	if !ok {
		t.Fatal("info/GetChild(id) not ok")
	}
}
