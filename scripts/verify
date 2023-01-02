#!/usr/bin/env bash
# Copyright (c) Berk D. Demir and the runitor contributors.
# SPDX-License-Identifier: 0BSD
set -euo pipefail

ALLOWED_SIGNERS_URL="https://bdd.fi/x/runitor.pub"
SIGNER_IDENTITY="runitor"
SIGNATURE_NAMESPACE="bdd.fi/x/runitor release"

SHA256SUM=$(type -p sha256sum shasum | head -1 || true)
if [[ -z ${SHA256SUM} ]]; then
  echo "couldn't find sha256sum or shasum in the PATH." >&2
  exit 69 # EX_UNAVAILABLE
fi

if ! hash ssh-keygen; then
  echo "need ssh-keygen to verify release signature." >&2
  exit 69 # EX_UNAVAILABLE
fi

if (($# > 1)); then
  echo "usage: $0 [releasedir]" >&2
  exit 64 # EX_USAGE
fi

releasedir="${1-${PWD}}"

(
  cd "${releasedir}"

  ssh-keygen -Y verify \
    -I "${SIGNER_IDENTITY}" \
    -n "${SIGNATURE_NAMESPACE}" \
    -f <(curl -Lsf "${ALLOWED_SIGNERS_URL}") \
    -s SHA256.sig < SHA256

  "${SHA256SUM}" --ignore-missing -c SHA256
)
