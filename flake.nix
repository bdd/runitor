# Copyright (c) Berk D. Demir and the runitor contributors.
# SPDX-License-Identifier: 0BSD
{
  description = "runitor";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs =
    { self, nixpkgs }:
    let
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      forSupportedSystems =
        f:
        nixpkgs.lib.genAttrs supportedSystems (system: {
          default = nixpkgs.legacyPackages.${system}.callPackage f { inherit self; };
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
    };
}
