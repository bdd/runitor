## Publish A Rebuild
If there's an update to Go either bringing in a bug for vulnerability fix we
release another build based on the original release tag.

Check out [Docker Scout Vulnerability View for
runitor](https://scout.docker.com/reports/org/runitor/vulnerabilities?stream=latest-indexed)
to see if we have CRITICAL or HIGH vulnerabilities.

Build and make a GitHub release:

```
nix develop --override-input nixpkgs github:NixOS/nixpkgs/nixpkgs-unstable

RELEASE=vX.Y.Z
RELBUILD=${RELEASE}-build.N

WORKTREE=$(git rev-parse --show-toplevel)/build/worktree/${RELEASE}
git worktree remove ${WORKTREE}
git worktree add ${WORKTREE} ${RELEASE}

GOTOOLCHAIN=go1.X.Y

export WORKTREE GOTOOLCHAIN
./scripts/mkrel ${RELBUILD}

# Cleanup
git worktree remove $WORKTREE
unset WORKTREE GOTOOLCHAIN
```

Then we build and publish the OCI images:

```
sudo podman run --privileged --rm tonistiigi/binfmt --install arm,arm64
mkdir -m 0700 -p ${XDG_RUNTIME_DIR}/containers
pass show host/$(hostname -s)/containers/auth.json > ${XDG_RUNTIME_DIR}/containers/auth.json

RELEASE=vX.Y.Z
RELBUILD=${RELEASE}-build.N
./scripts/mkoci build
./scripts/mkoci tag_rt_shorts
./scripts/mkoci tag_default_rt
./scripts/mkoci push

shred -u ${XDG_RUNTIME_DIR}/containers/auth.json
```

## Sign Unsigned Commits
Goes without saying this rewrites the history. So hopefully you're doing this
on a set of commits that never got pushed to the public remote.

With `git log --show-signature` find the SHA1 of the last signed commit, e.g. `0cafe42`.

```
ssh-add -K
git rebase --exec 'git commit --amend --no-edit -S' -i 0cafe42
```
