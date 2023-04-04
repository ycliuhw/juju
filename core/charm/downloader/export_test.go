// Copyright 2022 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package downloader

import (
	"github.com/juju/charm/v9"
	"gopkg.in/macaroon.v2"

	corecharm "github.com/juju/juju/core/charm"
)

func (dc DownloadedCharm) Verify(downloadOrigin corecharm.Origin, force bool) error {
	return dc.verify(downloadOrigin, force)
}

func (d *Downloader) NormalizePlatform(charmURL *charm.URL, platform corecharm.Platform) (corecharm.Platform, error) {
	return d.normalizePlatform(charmURL, platform)
}

func (d *Downloader) DownloadAndHash(charmURL *charm.URL, requestedOrigin corecharm.Origin, macaroons macaroon.Slice, repo CharmRepository, dstPath string) (DownloadedCharm, corecharm.Origin, error) {
	return d.downloadAndHash(charmURL, requestedOrigin, macaroons, repo, dstPath)
}
