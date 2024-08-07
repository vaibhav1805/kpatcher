class KPatcher < Formula
  desc "patch k8s resources at scale"
  homepage "https://github.com/vaibhav1805/kpatcher"
  version "1.0.0"

  if Hardware::CPU.intel?
    url "https://github.com/vaibhav1805/kpatcher/releases/download/v1.0.0/kpatcher-darwin-amd64.tar.gz"
    sha256 "72904e253dae842e3a41ef1d6d0a09fefcbfda70ef51aca5b8bf82685e34810f"
  elsif Hardware::CPU.arm?
    url "https://github.com/vaibhav1805/kpatcher/releases/download/v1.0.0/kpatcher-darwin-arm64.tar.gz"
    sha256 "8dc516682cf8ad725c19aa3ba0e51b2dee0b6f8615305ee02c934677ea4fa2c5"
  end

  depends_on "go" => :build

  def install
    if Hardware::CPU.intel?
      bin.install "kpatcher-darwin-amd64" => "kpatcher"
    elsif Hardware::CPU.arm?
      bin.install "kpatcher-darwin-arm64" => "kpatcher"
    end
  end

  test do
    assert_match "Usage", shell_output("#{bin}/kpatcher --help")
  end
end
