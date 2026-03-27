#!/bin/bash
# Copyright 2026 syzkaller project authors. All rights reserved.
# Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

# This script pulls pull request review comments from GitHub and saves them to a text file.

set -euo pipefail

# Target users (comma-separated list)
REVIEWERS="${3:-dvyukov}"

if [ -z "${1:-}" ]; then
    SINCE_DATE=$(date -d "-7 days" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null)
    if [ -z "$SINCE_DATE" ]; then
        # Fallback for non-GNU date
        SINCE_DATE="2026-03-20T00:00:00Z" # Approx (Today is Mar 27, so Mar 20 is 1 week ago)
    fi
else
    SINCE_DATE="$1"
fi

OUTPUT_FILE="${2:-dvyukov_comments.txt}"

echo "Pulling comments from users '${REVIEWERS}' since ${SINCE_DATE}..."
echo "Output file: ${OUTPUT_FILE}"

# Truncate output file before writing
> "${OUTPUT_FILE}"

# Fetch Pull Request Review comments across the repository.
# Pagination is handled by gh api --paginate.
# Use -X GET to ensure query parameters are appended correctly.
#
gh api "repos/google/syzkaller/pulls/comments" -X GET -F since="${SINCE_DATE}" --paginate | \
    jq -r --arg reviewers "$REVIEWERS" '.[] | select(.user.login | IN($reviewers | split(",")[])) | "\(.created_at)\t\(.html_url)\t\(.path)\t\(.user.login)\t\(.commit_id)\t\(.start_line // .original_start_line // "null")\t\(.line // .original_line // "null")\t\(.body | @base64)"' | \
    while IFS=$'\t' read -r DATE URL PATH_FILE USERNAME COMMIT_ID START_LINE END_LINE BASE64_BODY; do
        if [ "$START_LINE" = "null" ]; then
            START_LINE=$END_LINE
        fi
        if [ "$START_LINE" = "null" ]; then
            START_LINE=$END_LINE
        fi
        if [ "$START_LINE" = "null" ]; then
            START_LINE=1
            END_LINE=1
        fi

        CONTEXT_START=$((START_LINE - 3))
        if [ $CONTEXT_START -lt 1 ]; then
            CONTEXT_START=1
        fi
        CONTEXT_END=$((END_LINE + 3))

        echo "=== COMMENT START ===" >> "${OUTPUT_FILE}"
        echo "Date: ${DATE}" >> "${OUTPUT_FILE}"
        echo "User: ${USERNAME}" >> "${OUTPUT_FILE}"
        echo "URL: ${URL}" >> "${OUTPUT_FILE}"
        echo "File: ${PATH_FILE} (Lines: ${START_LINE}-${END_LINE}, Context: ${CONTEXT_START}-${CONTEXT_END})" >> "${OUTPUT_FILE}"

        CODE_LINE=""
        
        RAW_JSON=$(gh api "repos/google/syzkaller/contents/${PATH_FILE}" -X GET -F ref="${COMMIT_ID}" 2>/dev/null || true)
        if [ -n "$RAW_JSON" ]; then
            BASE64_FILE=$(echo "$RAW_JSON" | jq -r '.content')
            if [ "$BASE64_FILE" != "null" ]; then
                # decode and get range
                CODE_LINE=$(echo "$BASE64_FILE" | base64 -d 2>/dev/null | sed -n "${CONTEXT_START},${CONTEXT_END}p" || true)
            fi
        fi

        if [ -z "$CODE_LINE" ]; then
            CODE_LINE="[Could not fetch code line from GitHub API]"
        fi

        echo "Code lines:" >> "${OUTPUT_FILE}"
        echo "${CODE_LINE}" >> "${OUTPUT_FILE}"
        echo "" >> "${OUTPUT_FILE}"
        echo "Comment:" >> "${OUTPUT_FILE}"
        
        if command -v base64 >/dev/null 2>&1; then
            echo "$BASE64_BODY" | base64 -d >> "${OUTPUT_FILE}"
        else
            echo "[Base64 decoder missing, cannot print multiline body reliably]" >> "${OUTPUT_FILE}"
        fi
        
        echo "" >> "${OUTPUT_FILE}"
        echo "=== COMMENT END ===" >> "${OUTPUT_FILE}"
        echo "" >> "${OUTPUT_FILE}"
    done




echo "Finished. Extracted comments to ${OUTPUT_FILE}"
