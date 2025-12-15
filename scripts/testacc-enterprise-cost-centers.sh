#!/usr/bin/env sh
set -eu

# Run the GitHub Enterprise Cloud Cost Centers acceptance tests.
#
# Usage:
#   export GITHUB_TOKEN="..."   # classic PAT, enterprise admin
#   source scripts/env.enterprise-cost-centers.sh
#   scripts/testacc-enterprise-cost-centers.sh

require_env() {
  name="$1"
  if [ -z "${!name:-}" ]; then
    echo "Missing required env var: ${name}" 1>&2
    exit 1
  fi
}

require_env GITHUB_TOKEN
require_env ENTERPRISE_ACCOUNT
require_env ENTERPRISE_SLUG

# Only required for the authoritative membership resource test.
require_env ENTERPRISE_TEST_ORGANIZATION
require_env ENTERPRISE_TEST_REPOSITORY
require_env ENTERPRISE_TEST_USERS

# Run only the cost-centers acceptance tests.
TF_ACC=1 go test -v ./... -run '^TestAccGithubEnterpriseCostCenter'
