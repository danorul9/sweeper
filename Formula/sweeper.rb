class Sweeper < Formula
  desc "macOS app leftover detector & cleaner"
  homepage "https://github.com/danorul9/sweeper"
  version "0.3.1"
  license "MIT"

  on_macos do
    url "https://github.com/danorul9/sweeper/releases/download/v#{version}/sweeper-#{version}-darwin-all"
    sha256 "c38e7d479dd6207fe41cae461aa1f3ff6ab34fb67aeddcc70616342cf52655f8"
  end

  def install
    bin.install "sweeper-#{version}-darwin-all" => "sweeper"
  end

  test do
    system "#{bin}/sweeper", "--help"
  end
end
