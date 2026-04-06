# frozen_string_literal: true

# Homebrew formula for the Node.js + Ink CLI (replaces the former Go/cask binary).
# After each npm publish, refresh the sha256:
#   curl -sL "$(npm view topoductor dist.tarball)" | shasum -a 256
# and bump the `url` version segment if needed.

class Topoductor < Formula
  desc "Terminal UI for git worktrees"
  homepage "https://github.com/brandonhsz/TopoDuctor"
  url "https://registry.npmjs.org/topoductor/-/topoductor-0.2.0.tgz"
  sha256 "20de7fb1f064176b82ea9501686e1a3ff9e36c0487a878fe6d605ed600481cd9"
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
