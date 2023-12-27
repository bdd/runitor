# Copyright (c) Berk D. Demir and the runitor contributors.
# SPDX-License-Identifier: 0BSD
{
  description = "runitor";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-23.11-darwin";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }: flake-utils.lib.eachDefaultSystem (system:
    let
      pkgs = import nixpkgs { inherit system; };
      runitor = pkgs.buildGoModule rec {
        pname = "runitor";
        revDate = builtins.substring 0 8 (self.lastModifiedDate or "19700101");
        version = "${revDate}-${self.shortRev or "dirty"}";
        vendorHash = null;
        src = ./.;
        CGO_ENABLED = 0;
        ldflags = [ "-s" "-w" "-X main.Version=v${version}" ];
        meta = {
          homepage = "https://bdd.fi/x/runitor";
          description = "A command runner with healthchecks.io integration";
          longDescription = ''
            Runitor runs the supplied command, captures its output, and based on its exit
            code reports successful or failed execution to https://healthchecks.io or your
            private instance.

            Healthchecks.io is a web service for monitoring periodic tasks. It's like a
            dead man's switch for your cron jobs. You get alerted if they don't run on time
            or terminate with a failure.
          '';
          license = pkgs.lib.licenses.bsd0;
          mainProgram = "runitor";
          maintainers = [ pkgs.maintainers.bdd ];
        };
      };

      enumer = pkgs.buildGoModule rec {
        pname = "enumer";
        version = "1.5.9";
        src = pkgs.fetchFromGitHub {
          owner = "dmarkham";
          repo = "enumer";
          rev = "v${version}";
          hash = "sha256-NYL36GBogFM48IgIWhFa1OLZNUeEi0ppS6KXybnPQks=";
        };
        vendorHash = "sha256-CJCay24FlzDmLjfZ1VBxih0f+bgBNu+Xn57QgWT13TA=";
        meta = {
          description = "A Go tool to auto generate methods for your enums";
          license = pkgs.lib.licenses.bsd2;
          mainProgram = "enumer";
          maintainers = [ pkgs.maintainers.bdd ];
        };
      };
    in
    {
      devShells = {
        default = pkgs.mkShell {
          buildInputs = [
            # build
            pkgs.go
            self.packages.${system}.enumer

            # release
            pkgs.gh # create a release on github and upload artifacts
            pkgs.git # mkrel: git tag, git push
            pkgs.curl # verify, dlrel, build
            pkgs.coreutils # sha256sum: sign & verify
            pkgs.openssh # ssh-keygen: sign & verify
          ];
        };
      };

      packages = {
        inherit runitor enumer;
        default = runitor;
      };
    }
  );
}
