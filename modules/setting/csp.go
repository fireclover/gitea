// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

// CSPConfig defines CSP settings
var CSPConfig = struct {
	Enabled          bool
	ReportOnly 		 bool
	Directives       []string
}{
	Enabled:   	   false,
	ReportOnly:    false,
}

func loadCspFrom(rootCfg ConfigProvider) {
	mustMapSetting(rootCfg, "csp", &CSPConfig)
}