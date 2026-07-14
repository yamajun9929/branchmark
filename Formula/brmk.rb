class Brmk < Formula
  desc "Branchmark, a low-dependency terminal bookmark tree"
  homepage "https://github.com/yamajun9929/branchmark"
  url "https://github.com/yamajun9929/branchmark/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "87004303a3148e72ed6e659c0add6b9a88ced110fa024603bc40b0b82f2396c7"
  license "MIT"

  depends_on "go" => :build

  def install
    ldflags = "-s -w -X main.version=#{version}"
    system "go", "build", "-trimpath", "-ldflags", ldflags, "-o", bin/"brmk", "./cmd/brmk"
    generate_completions_from_executable(bin/"brmk", "completion")
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/brmk version")
  end
end
