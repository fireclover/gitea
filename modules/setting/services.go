// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"fmt"
	"os"
	"path/filepath"
)

// Service registry settings
var (
	Services = struct {
		Storage           *Storage
		Enabled           bool
		ChunkedUploadPath string

		LimitTotalOwnerCount int64
		LimitTotalOwnerSize  int64
		LimitSizeAlpine      int64
		LimitSizeArch        int64
		LimitSizeCargo       int64
		LimitSizeChef        int64
		LimitSizeComposer    int64
		LimitSizeConan       int64
		LimitSizeConda       int64
		LimitSizeContainer   int64
		LimitSizeCran        int64
		LimitSizeDebian      int64
		LimitSizeGeneric     int64
		LimitSizeGo          int64
		LimitSizeHelm        int64
		LimitSizeMaven       int64
		LimitSizeNpm         int64
		LimitSizeNuGet       int64
		LimitSizePub         int64
		LimitSizePyPI        int64
		LimitSizeRpm         int64
		LimitSizeRubyGems    int64
		LimitSizeSwift       int64
		LimitSizeVagrant     int64

		DefaultRPMSignEnabled bool
	}{
		Enabled:              true,
		LimitTotalOwnerCount: -1,
	}
)

func loadServicesFrom(rootCfg ConfigProvider) (err error) {
	sec, _ := rootCfg.GetSection("services")
	if sec == nil {
		Services.Storage, err = getStorage(rootCfg, "services", "", nil)
		return err
	}

	if err = sec.MapTo(&Services); err != nil {
		return fmt.Errorf("failed to map Services settings: %v", err)
	}

	Services.Storage, err = getStorage(rootCfg, "services", "", sec)
	if err != nil {
		return err
	}

	Services.ChunkedUploadPath = filepath.ToSlash(sec.Key("CHUNKED_UPLOAD_PATH").MustString("tmp/service-upload"))
	if !filepath.IsAbs(Services.ChunkedUploadPath) {
		Services.ChunkedUploadPath = filepath.ToSlash(filepath.Join(AppDataPath, Services.ChunkedUploadPath))
	}

	if HasInstallLock(rootCfg) {
		if err := os.MkdirAll(Services.ChunkedUploadPath, os.ModePerm); err != nil {
			return fmt.Errorf("unable to create chunked upload directory: %s (%v)", Services.ChunkedUploadPath, err)
		}
	}

	Services.LimitTotalOwnerSize = mustBytes(sec, "LIMIT_TOTAL_OWNER_SIZE")
	Services.LimitSizeAlpine = mustBytes(sec, "LIMIT_SIZE_ALPINE")
	Services.LimitSizeArch = mustBytes(sec, "LIMIT_SIZE_ARCH")
	Services.LimitSizeCargo = mustBytes(sec, "LIMIT_SIZE_CARGO")
	Services.LimitSizeChef = mustBytes(sec, "LIMIT_SIZE_CHEF")
	Services.LimitSizeComposer = mustBytes(sec, "LIMIT_SIZE_COMPOSER")
	Services.LimitSizeConan = mustBytes(sec, "LIMIT_SIZE_CONAN")
	Services.LimitSizeConda = mustBytes(sec, "LIMIT_SIZE_CONDA")
	Services.LimitSizeContainer = mustBytes(sec, "LIMIT_SIZE_CONTAINER")
	Services.LimitSizeCran = mustBytes(sec, "LIMIT_SIZE_CRAN")
	Services.LimitSizeDebian = mustBytes(sec, "LIMIT_SIZE_DEBIAN")
	Services.LimitSizeGeneric = mustBytes(sec, "LIMIT_SIZE_GENERIC")
	Services.LimitSizeGo = mustBytes(sec, "LIMIT_SIZE_GO")
	Services.LimitSizeHelm = mustBytes(sec, "LIMIT_SIZE_HELM")
	Services.LimitSizeMaven = mustBytes(sec, "LIMIT_SIZE_MAVEN")
	Services.LimitSizeNpm = mustBytes(sec, "LIMIT_SIZE_NPM")
	Services.LimitSizeNuGet = mustBytes(sec, "LIMIT_SIZE_NUGET")
	Services.LimitSizePub = mustBytes(sec, "LIMIT_SIZE_PUB")
	Services.LimitSizePyPI = mustBytes(sec, "LIMIT_SIZE_PYPI")
	Services.LimitSizeRpm = mustBytes(sec, "LIMIT_SIZE_RPM")
	Services.LimitSizeRubyGems = mustBytes(sec, "LIMIT_SIZE_RUBYGEMS")
	Services.LimitSizeSwift = mustBytes(sec, "LIMIT_SIZE_SWIFT")
	Services.LimitSizeVagrant = mustBytes(sec, "LIMIT_SIZE_VAGRANT")
	Services.DefaultRPMSignEnabled = sec.Key("DEFAULT_RPM_SIGN_ENABLED").MustBool(false)
	return nil
}

