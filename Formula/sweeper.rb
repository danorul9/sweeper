class Sweeper < Formula
  desc "macOS app leftover detector & cleaner"
  homepage "https://github.com/danorul9/sweeper"
  version "0.5.0"
  license "MIT"

  on_macos do
    url "https://github.com/danorul9/sweeper/releases/download/v#{version}/sweeper-#{version}-darwin-all"
    sha256 "a98bbbb0b91c40e0a3f816d387b19fef061844be7770d60644921d428bc2a3f6"
  end

  def install
    bin.install "sweeper-#{version}-darwin-all" => "sweeper"
  end

  test do
    system "#{bin}/sweeper", "--help"
  end
end
