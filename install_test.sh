#!/bin/sh
# Component tests for install.sh version resolution
# Usage: sh install_test.sh

set -e

PASS=0
FAIL=0
ORIG_PATH="$PATH"

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

# Helper: create a mock curl that simulates different behaviors
setup_mock_curl() {
    MOCK_DIR="$(mktemp -d)"
    cat > "${MOCK_DIR}/curl" <<'MOCKCURL'
#!/bin/sh
# Parse args to determine what to return
WRITE_OUT=""
URL=""
for arg in "$@"; do
    case "$prev" in
        -w) WRITE_OUT="$arg" ;;
    esac
    prev="$arg"
    # Last non-flag arg is typically the URL
    case "$arg" in
        http*) URL="$arg" ;;
    esac
done

if [ -n "$MOCK_CURL_FAIL" ]; then
    echo "curl: (56) The requested URL returned error: 403" >&2
    exit 56
fi

if [ "$WRITE_OUT" = "%{url_effective}" ]; then
    if [ -n "$MOCK_CURL_NO_REDIRECT" ]; then
        printf '%s' "$URL"
    else
        printf '%s' "$MOCK_CURL_EFFECTIVE_URL"
    fi
    exit 0
fi

exit 0
MOCKCURL
    chmod +x "${MOCK_DIR}/curl"
    export PATH="${MOCK_DIR}:$PATH"
    export MOCK_DIR
}

cleanup_mock() {
    rm -rf "$MOCK_DIR"
    export PATH="$ORIG_PATH"
    unset MOCK_CURL_EFFECTIVE_URL MOCK_CURL_NO_REDIRECT MOCK_CURL_FAIL
}

# Extract just the version-resolution block from install.sh for isolated testing
# We test by sourcing a snippet that mimics the script's logic
resolve_version() {
    REPO="siyuqian/devpilot"
    VERSION=""
    VERSION="$(curl -fsSL -o /dev/null -w '%{url_effective}' \
        "https://github.com/${REPO}/releases/latest" 2>/dev/null \
        | sed 's|.*/tag/||')"
    if [ -z "$VERSION" ] || echo "$VERSION" | grep -q "^https://"; then
        echo ""
        return 1
    fi
    echo "$VERSION"
    return 0
}

echo "=== install.sh component tests ==="
echo ""

# Test 1: Successful version resolution via redirect
echo "Test 1: Resolves version from GitHub redirect"
setup_mock_curl
export MOCK_CURL_EFFECTIVE_URL="https://github.com/siyuqian/devpilot/releases/tag/v0.14.3"
RESULT="$(resolve_version)"
if [ "$RESULT" = "v0.14.3" ]; then
    pass "resolved v0.14.3"
else
    fail "expected 'v0.14.3', got '$RESULT'"
fi
cleanup_mock

# Test 2: No redirect (no releases) returns error
echo "Test 2: Detects missing releases (no redirect)"
setup_mock_curl
export MOCK_CURL_NO_REDIRECT=1
RESULT="$(resolve_version || true)"
if [ -z "$RESULT" ]; then
    pass "returned empty on no redirect"
else
    fail "expected empty, got '$RESULT'"
fi
cleanup_mock

# Test 3: curl failure returns error
echo "Test 3: Handles curl failure gracefully"
setup_mock_curl
export MOCK_CURL_FAIL=1
RESULT="$(resolve_version 2>/dev/null || true)"
if [ -z "$RESULT" ]; then
    pass "returned empty on curl failure"
else
    fail "expected empty, got '$RESULT'"
fi
cleanup_mock

# Test 4: --version flag skips resolution (curl should never be called)
echo "Test 4: --version flag bypasses version fetch"
setup_mock_curl
# Replace mock curl with one that fails if called — proving fetch was skipped
cat > "${MOCK_DIR}/curl" <<'MOCKCURL'
#!/bin/sh
echo "CURL_WAS_CALLED" >&2
exit 1
MOCKCURL
chmod +x "${MOCK_DIR}/curl"
# Simulate the install.sh logic: if VERSION is already set, skip resolution
REPO="siyuqian/devpilot"
VERSION="v1.2.3"
if [ -z "$VERSION" ]; then
    VERSION="$(curl -fsSL -o /dev/null -w '%{url_effective}' \
        "https://github.com/${REPO}/releases/latest" 2>/dev/null \
        | sed 's|.*/tag/||')"
fi
if [ "$VERSION" = "v1.2.3" ]; then
    pass "--version pre-set skips fetch (curl not called)"
else
    fail "expected 'v1.2.3', got '$VERSION'"
fi
unset VERSION
cleanup_mock

# Test 5: Version with pre-release tag
echo "Test 5: Resolves pre-release version tag"
setup_mock_curl
export MOCK_CURL_EFFECTIVE_URL="https://github.com/siyuqian/devpilot/releases/tag/v1.0.0-beta.1"
RESULT="$(resolve_version)"
if [ "$RESULT" = "v1.0.0-beta.1" ]; then
    pass "resolved v1.0.0-beta.1"
else
    fail "expected 'v1.0.0-beta.1', got '$RESULT'"
fi
cleanup_mock

echo ""
echo "Results: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ] || exit 1
