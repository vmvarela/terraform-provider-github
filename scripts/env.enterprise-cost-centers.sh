#!/usr/bin/env sh

# NOTE: This file is meant to be sourced (not executed). Avoid `set -e`/`set -u`
# here because they would leak into the caller shell and can terminate
# interactive terminals (e.g. zsh prompts/plugins) unexpectedly.

# Environment variables for running the GitHub Enterprise Cloud Cost Centers acceptance tests.
#
# Usage:
#   source scripts/env.enterprise-cost-centers.sh
#   TF_ACC=1 TF_LOG=DEBUG go test -v ./... -run '^TestAccGithubEnterpriseCostCenter'
#
# Notes:
# - These endpoints require an enterprise admin using a classic PAT.
# - GitHub App tokens and fine-grained personal access tokens are not supported.

# Required by enterprise-mode acceptance tests.
export ENTERPRISE_ACCOUNT="true"
export ENTERPRISE_SLUG="prisa-media-emu"

# Fixtures used by the authoritative membership resource tests.
export ENTERPRISE_TEST_ORGANIZATION="PrisaMedia-Training-Sandbox"
export ENTERPRISE_TEST_REPOSITORY="PrisaMedia-Training-Sandbox/vmvarela-testing-cost-centers"
export ENTERPRISE_TEST_USERS="vmvarela-clb_prisa,ebarberan-clb_prisa"

# Classic personal access token (PAT) for an enterprise admin.
# IMPORTANT: do not commit real tokens.
: "${GITHUB_TOKEN:=}"
export GITHUB_TOKEN
