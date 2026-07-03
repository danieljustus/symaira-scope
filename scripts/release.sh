#!/bin/bash
set -euo pipefail

# Symscope - Release Script
# Usage: ./scripts/release.sh <version>
# Example: ./scripts/release.sh 0.1.3

VERSION="${1:?Usage: $0 <version>}"
APP_NAME="Symscope"
SCHEME="Symscope"
BUILD_DIR="client/build/release"
ARCHIVE_PATH="${BUILD_DIR}/${APP_NAME}.xcarchive"
EXPORT_PATH="${BUILD_DIR}/export"
DMG_PATH="${BUILD_DIR}/${APP_NAME}-${VERSION}.dmg"

echo "=== Building ${APP_NAME} ${VERSION} ==="

rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

# Generate Xcode project
(cd client && xcodegen generate)

# Check if signed release is requested
SIGNED=false
if [ "${2:-}" = "--signed" ] || [ -n "${CODESIGN_IDENTITY:-}" ]; then
    if security find-identity -v -p codesigning | grep -q "Developer ID Application"; then
        echo "Developer ID Application certificate found. Building signed release."
        SIGNED=true
    else
        echo "Warning: Signed release requested, but no Developer ID Application certificate found in keychain. Building unsigned."
    fi
else
    echo "Building unsigned/ad-hoc release. (Pass --signed or set CODESIGN_IDENTITY for a signed build)"
fi

# Set Xcode developer dir if Xcode-beta is present
if [ -d "/Applications/Xcode-beta.app" ]; then
    export DEVELOPER_DIR="/Applications/Xcode-beta.app/Contents/Developer"
fi

if [ "$SIGNED" = true ]; then
    echo "Archiving..."
    BUILD_NUMBER=$(git rev-list --count HEAD)
    xcodebuild archive \
        -project "client/${APP_NAME}.xcodeproj" \
        -scheme "${SCHEME}" \
        -archivePath "${ARCHIVE_PATH}" \
        -configuration Release \
        MARKETING_VERSION="${VERSION}" \
        CURRENT_PROJECT_VERSION="${BUILD_NUMBER}" \
        CODE_SIGN_IDENTITY="Developer ID Application" \
        CODE_SIGN_STYLE=Manual \
        CODE_SIGNING_ALLOWED=YES
    
    echo "Exporting..."
    cat > "${BUILD_DIR}/ExportOptions.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>method</key>
    <string>developer-id</string>
    <key>signingStyle</key>
    <string>manual</string>
    <key>signingCertificate</key>
    <string>Developer ID Application</string>
</dict>
</plist>
EOF

    xcodebuild -exportArchive \
        -archivePath "${ARCHIVE_PATH}" \
        -exportOptionsPlist "${BUILD_DIR}/ExportOptions.plist" \
        -exportPath "${EXPORT_PATH}"
        
    APP_PATH=$(find "${EXPORT_PATH}" -name "*.app" -type d | head -1)
else
    echo "Building (unsigned)..."
    BUILD_NUMBER=$(git rev-list --count HEAD)
    xcodebuild build \
        -project "client/${APP_NAME}.xcodeproj" \
        -scheme "${SCHEME}" \
        -configuration Release \
        -derivedDataPath "${BUILD_DIR}/DerivedData" \
        MARKETING_VERSION="${VERSION}" \
        CURRENT_PROJECT_VERSION="${BUILD_NUMBER}" \
        CODE_SIGN_IDENTITY="" \
        CODE_SIGNING_REQUIRED=NO \
        CODE_SIGNING_ALLOWED=NO
        
    # Find the built .app
    APP_PATH=$(find "${BUILD_DIR}/DerivedData" -name "${APP_NAME}.app" -type d | head -1)
    mkdir -p "${EXPORT_PATH}"
    cp -R "$APP_PATH" "${EXPORT_PATH}/"
    APP_PATH="${EXPORT_PATH}/${APP_NAME}.app"
fi

if [ -z "${APP_PATH:-}" ] || [ ! -d "${APP_PATH}" ]; then
    echo "Error: No .app found in build paths"
    exit 1
fi

echo "Creating DMG..."
hdiutil create -volname "${APP_NAME}" \
    -srcfolder "${APP_PATH}" \
    -ov -format UDZO \
    "${DMG_PATH}"

echo "Uploading to GitHub..."
# If the release already exists, upload the DMG to it, otherwise create it.
if gh release view "v${VERSION}" >/dev/null 2>&1; then
    echo "Release v${VERSION} already exists. Uploading DMG asset."
    gh release upload "v${VERSION}" \
        "${DMG_PATH}#${APP_NAME}-${VERSION}.dmg" \
        --clobber
else
    echo "Creating new release v${VERSION} and uploading DMG."
    gh release create "v${VERSION}" \
        "${DMG_PATH}#${APP_NAME}-${VERSION}.dmg" \
        --title "v${VERSION}" \
        --generate-notes
fi

echo "=== Release ${VERSION} complete ==="
