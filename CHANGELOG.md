<!-- SPDX-License-Identifier: Apache-2.0 -->
<!-- Copyright (C) 2026 Ross Video. All rights reserved. -->

# Downstream changelog

## [Unreleased]

### Fixed

- Skip deletion of nonexistent ownership TXT records when deleting DNS records, including AWS A ALIAS records
  still owned through legacy `cname-` markers. This prevents Route53 from rejecting the deletion batch because
  the new `a-` ownership TXT has not been created. Legacy-marker cleanup and ownership selection are unchanged.
- Preserve the TXT inventory on cached reads and refresh the registry snapshot after writes, including failed
  writes, so subsequent deletions use current TXT existence information. With TXT caching enabled, the next
  reconciliation after a write now reads the provider instead of reusing the locally amended snapshot.
