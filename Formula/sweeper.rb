class Sweeper < Formula
  desc "macOS app leftover detector & cleaner"
  homepage "https://github.com/danorul9/sweeper"
  version "0.4.0"
  license "MIT"

  on_macos do
    url "https://github.com/danorul9/sweeper/releases/download/v#{version}/sweeper-#{version}-darwin-all"
    sha256 "acfaa914a3bb891ecb106726bc0582a2fd3d593bc0ba012f438d2d69af0f36c6"
  end

  def install
    bin.install "sweeper-#{version}-darwin-all" => "sweeper"
  end

  test do
    system "#{bin}/sweeper", "--help"
  end
end
