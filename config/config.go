// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"io/ioutil"

	"github.com/BurntSushi/toml"
)

func LoadConfig() (*Config, error) {
	data, err := ioutil.ReadFile("./sbrwxmpp.toml")
	if err != nil {
		return nil, err
	}
	config := new(Config)
	_, err = toml.Decode(string(data), config)
	return config, err
}
