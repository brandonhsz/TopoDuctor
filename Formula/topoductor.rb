# frozen_string_literal: true

# Homebrew formula for the Node.js + Ink CLI (replaces the former Go/cask binary).
# After each npm publish, refresh the sha256:
#   curl -sL "$(npm view topoductor dist.tarball)" | shasum -a 256
# and bump the `url` version segment if needed.

class Topoductor < Formula
  desc "Terminal UI for git worktrees"
  homepage "https://github.com/brandonhsz/TopoDuctor"
  url "https://registry.npmjs.org/topoductor/-/topoductor-0.0.1.tgz"
  sha256 "903dcfb50d527550f490eab889e6db4bde88136d55e33aa9823f4c7a4e020862"
  license "MIT"

  depends_on "node"

  def install
    system "npm", "install", *Language::Node.std_npm_install_args(libexec)
    bin.install_symlink Dir["#{libexec}/bin/*"]
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/topoductor --version")
  end
end
