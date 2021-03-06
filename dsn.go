// Copyright 2019 The SQLFlow Authors. All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package goalisa

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	reDSN = regexp.MustCompile(`^([a-zA-Z0-9_-]+):([=a-zA-Z0-9_-]+)@([:a-zA-Z0-9/_.-]+)\?([^/]+)$`)
)

// Config is the deserialization of connect string, the connection string should of format:
// pop_access_id:pop_access_secret@pop_url?env=..
type Config struct {
	POPAccessID     string            // POP config
	POPAccessSecret string            // POP config
	POPURL          string            // POPURL does not contain a header like: http/https
	POPScheme       string            // POPScheme: http/https
	Env             map[string]string // Environment variable JSON encoded in base64 format.
	With            map[string]string // With variable JSON encoded in base64 format.
	Verbose         bool              // Verbose denotes whether to print logs to the terminal
	Project         string            // Project(name) of Alisa, generated from Env.
}

// ParseDSN deserialize the connect string
func ParseDSN(dsn string) (*Config, error) {
	sub := reDSN.FindStringSubmatch(dsn)
	if len(sub) != 5 {
		return nil, fmt.Errorf(`dsn %s doesn't match pop_access_id:pop_access_secret@pop_url?params`, dsn)
	}
	pid, ps, purl := sub[1], sub[2], sub[3]

	kvs, err := url.ParseQuery(sub[4])
	if err != nil {
		return nil, err
	}

	requiredParameter := []string{"env", "with", "curr_project"}
	for _, k := range requiredParameter {
		v := kvs.Get(k)
		if v == "" {
			return nil, fmt.Errorf(`dsn is missing required parameter %s`, k)
		}
	}

	env, err := decodeJSONB64(kvs.Get("env"))
	if err != nil {
		return nil, err
	}
	with, err := decodeJSONB64(kvs.Get("with"))
	if err != nil {
		return nil, err
	}

	scheme := kvs.Get("scheme")
	if len(scheme) == 0 {
		scheme = "http"
	}
	verbose := kvs.Get("verbose") == "true"
	project := kvs.Get("curr_project")

	return &Config{POPAccessID: pid, POPAccessSecret: ps, POPURL: purl, Env: env, With: with, Verbose: verbose, Project: project, POPScheme: scheme}, nil
}

func encodeJSONB64(env map[string]string) string {
	// We sort the env params to ensure the consistent encoding
	keys := []string{}
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	kvStrs := []string{}
	for _, k := range keys {
		kvStrs = append(kvStrs, fmt.Sprintf(`"%s":"%s"`, k, env[k]))
	}
	jsonStr := `{` + strings.Join(kvStrs, `,`) + `}`
	return base64.RawURLEncoding.EncodeToString([]byte(jsonStr))
}

func decodeJSONB64(b64env string) (map[string]string, error) {
	// NOTE(tony): we use url.ParseQuery to parse parameters in ParseDSN,
	// so we use URL-compatible base64 format.
	buf, err := base64.RawURLEncoding.DecodeString(b64env)
	if err != nil {
		return nil, err
	}
	var envs map[string]string
	if err := json.Unmarshal(buf, &envs); err != nil {
		return nil, err
	}
	return envs, nil
}

// FormatDSN serialize a config to connect string
func (cfg *Config) FormatDSN() string {
	return fmt.Sprintf(`%s:%s@%s?env=%s&with=%s&verbose=%s&curr_project=%s&scheme=%s`,
		cfg.POPAccessID, cfg.POPAccessSecret, cfg.POPURL, encodeJSONB64(cfg.Env),
		encodeJSONB64(cfg.With), strconv.FormatBool(cfg.Verbose), cfg.Project, cfg.POPScheme)
}
