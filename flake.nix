# Copyright (c) Berk D. Demir and the runitor contributors.
# SPDX-License-Identifier: 0BSD
{
  description = "runitor";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs =
    { self, nixpkgs }:
    let
      inherit (nixpkgs.lib) mapCartesianProduct genAttrs;

      supportedSystems = mapCartesianProduct ({ arch, os }: "${arch}-${os}") {
        arch = [ "x86_64" "aarch64" ];
        os = [ "linux" "darwin" ];
      };

      forSupportedSystems = pkg: genAttrs supportedSystems (system: {
        default = nixpkgs.legacyPackages.${system}.callPackage pkg { inherit self; };
      });
    in
    {
      devShells = forSupportedSystems (
        { pkgs, ... }:

        pkgs.mkShell {
          buildInputs = with pkgs; [
            # build
            go

            # release
            gh # create a release on github and upload artifacts
            git # mkrel: git tag, git push
            curl # verify, dlrel, build
            coreutils # sha256sum: sign & verify
            openssh # ssh-keygen: sign & verify
          ];
        }
      );

      packages = forSupportedSystems ./package.nix;

      formatter = genAttrs supportedSystems (system: nixpkgs.legacyPackages.${system}.nixpkgs-fmt);
    };
}
