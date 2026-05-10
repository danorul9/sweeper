class Sweeper < Formula
  desc "macOS app leftover detector & cleaner"
  homepage "https://github.com/danorul9/sweeper"
  version "0.1.0"
  license "MIT"

  on_macos do
    url "https://github.com/danorul9/sweeper/releases/download/v#{version}/sweeper-#{version}-darwin-all"
    sha256 "2d0516abd0ec90222b4b00c350f450cce1a0dc3bf53f940ea72333c713a80833"
  end

  def install
    bin.install "sweeper-#{version}-darwin-all" => "sweeper"
  end

  test do
    system "#{bin}/sweeper", "--help"
  end
end
