#!/bin/sh

set -eu

root_dir=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
workflow="$root_dir/.github/workflows/release.yml"

fail() {
  printf '%s\n' "$*" >&2
  exit 1
}

grep -Fq 'tag: ${{ steps.validated_tag.outputs.tag }}' "$workflow" || fail 'release workflow must publish the validated tag output'
grep -Fq 'RELEASE_TAG_INPUT: ${{ github.event_name == '\''workflow_dispatch'\'' && github.event.inputs.tag || github.ref_name }}' "$workflow" || fail 'release workflow must pass the dispatch tag through an environment variable'
grep -Fq 'TAG_NAME="$RELEASE_TAG_INPUT"' "$workflow" || fail 'release workflow must read the dispatch tag from its environment'
grep -Fq 'ref: ${{ needs.validate-release-tag.outputs.tag }}' "$workflow" || fail 'release checkout must use the validated tag output'

# build-frontend must depend on validate-release-tag and check out the validated tag
sed -n '/^  build-frontend:/{n;p}' "$workflow" | grep -Fq '    needs: [validate-release-tag]' || fail 'build-frontend must depend on validate-release-tag'
# Verify build-frontend checkout uses the validated tag (not the default branch)
sed -n '/^  build-frontend:/,/^  release:/p' "$workflow" | grep -Fq 'ref: ${{ needs.validate-release-tag.outputs.tag }}' || fail 'build-frontend checkout must use the validated tag output'

if grep -Fq 'TAG_NAME=${{ github.event.inputs.tag }}' "$workflow"; then
  fail 'release workflow must not interpolate dispatch input directly into shell source'
fi
if grep -Fq "TAG_MESSAGE='\${{ steps.tag_message.outputs.message }}'" "$workflow"; then
  fail 'release workflow must not interpolate tag messages directly into shell source'
fi

printf 'release workflow security checks passed\n'
