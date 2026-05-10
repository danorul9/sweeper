class Sweeper < Formula
  desc "macOS app leftover detector & cleaner"
  homepage "https://github.com/danorul9/sweeper"
  version "{{.Version}}"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/danorul9/sweeper/releases/download/v#{version}/sweeper-v#{version}-darwin-arm64"
      sha256 "{{.SHA256}}"
    end
    on_intel do
      url "https://github.com/danorul9/sweeper/releases/download/v#{version}/sweeper-v#{version}-darwin-amd64"
      sha256 "{{.SHA256}}"
    end
  end

  def install
    bin.install "sweeper"
  end

  test do
    system "#{bin}/sweeper", "--help"
  end
end
