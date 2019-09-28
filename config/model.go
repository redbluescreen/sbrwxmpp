// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package config

type Config struct {
	Addr    string
	Cert    string
	CertKey string
	Domain  string
	API     APIConfig
	Webhook WebhookConfig
	Logging map[string]LoggingCategory
}

type LoggingCategory struct {
	Destination string
}

type APIConfig struct {
	Addr string
	Key  string
}

type WebhookConfig struct {
	Target string
	Secret string
}
