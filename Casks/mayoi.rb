cask "mayoi" do
  arch arm: "arm64", intel: "amd64"

  version "0.12.2"
  sha256 arm:   "b9a14320f53fcae052188030febc2717e5eaedbbba988d4023ce2bb7923d20ec",
         intel: "9bc4e051fb44cc560d3f20a90e10c161ec8c5fbf0632977cefd71546ea204743"

  url "https://github.com/sorrycc/Mayoi/releases/download/v#{version}/Mayoi.#{version}.#{arch}.dmg"
  name "Mayoi"
  desc "Small, fast code editor"
  homepage "https://github.com/sorrycc/Mayoi"

  auto_updates true
  depends_on macos: :ventura

  app "Mayoi.app"
  command_wrapper "mayoi", content: <<~SH
    #!/bin/sh
    # Managed by Homebrew; use brew uninstall --cask mayoi to remove it.
    app=#{(appdir/"Mayoi.app/Contents/MacOS/Mayoi").to_s.shellescape}
    [ -x "$app" ] || { echo "mayoi: Mayoi.app is not found" >&2; exit 1; }
    MAYOI_CLI=1 exec "$app" "$@"
  SH
end
