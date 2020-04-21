#!/usr/bin/env bash
set -euo pipefail

# You might be wondering why this script exists.
# The author has a monolithic repository
# (https://en.wikipedia.org/wiki/Monorepo). What you see as this repository
# actually is just a directory under another private repository. This script
# cherry-picks commits affecting files under that directory and rewrites them
# for this repo by dropping the subtree path.

remote=src
rmaster=${remote}/master
first_commit=fabf1579d0f1b32f33f136bd0f5d1bf3158067b1
subtree=hcpingrun

git fetch ${remote}
for commit in $(git log --reverse --pretty=%H ${rmaster} ${first_commit}...${rmaster}); do
  git cherry-pick -Xsubtree=${subtree} -x "${commit}"
done
