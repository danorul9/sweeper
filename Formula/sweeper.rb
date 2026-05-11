class Sweeper < Formula
  desc "macOS app leftover detector & cleaner"
  homepage "https://github.com/danorul9/sweeper"
  version "0.3.0"
  license "MIT"

  on_macos do
    url "https://github.com/danorul9/sweeper/releases/download/v#{version}/sweeper-#{version}-darwin-all"
    sha256 "7daadb56759549e07b4f0a8ef8120067531f17188eaf6dc21f29c66f69925db4"
  end

  def install
    bin.install "sweeper-#{version}-darwin-all" => "sweeper"
  end

  test do
    system "#{bin}/sweeper", "--help"
  end
end
