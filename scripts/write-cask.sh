#!/bin/bash
# Usage: write-cask.sh VERSION SHA OUTPUT_FILE
# Generates the Homebrew cask for Setzer (pushed to crux/homebrew-tap by CI).
set -e
VERSION=$1
SHA=$2
OUTPUT=$3

cat > "$OUTPUT" << CASK
cask "setzer" do
  version "${VERSION}"
  sha256 "${SHA}"
  url "https://github.com/crux/setzer/releases/download/v#{version}/Setzer-#{version}.dmg"

  name "Setzer"
  desc "Local compositor for static sites — edit and publish via Git"
  homepage "https://github.com/crux/setzer"

  app "Setzer.app"

  postflight do
    system_command "/usr/bin/xattr",
      args: ["-rd", "com.apple.quarantine", "#{appdir}/Setzer.app"]
  end
end
CASK
