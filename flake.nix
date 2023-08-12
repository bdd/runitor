# Copyright (c) Berk D. Demir and the runitor contributors.
# SPDX-License-Identifier: 0BSD
{
  description = "runitor";

  inputs = {
    nixpkgs.url = "github:bdd/nixpkgs/go_1_21";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }: flake-utils.lib.eachDefaultSystem (system:
    let
      pkgs = import nixpkgs { inherit system; };
    in
    {
      devShells = {
        default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # build
            go_1_21
            self.packages.${system}.enumer

            # release
            gh # create a release on github and upload artifacts
            git # mkrel: git tag, git push
            curl # verify, dlrel, build
            coreutils # sha256sum: sign & verify
            openssh # ssh-keygen: sign & verify
          ];
        };
      };

      packages = with pkgs; {
        default = buildGo121Module rec {
          pname = "runitor";
          revDate = builtins.substring 0 8 (self.lastModifiedDate or "19700101");
          version = "${revDate}-${self.shortRev or "dirty"}";
          vendorSha256 = null;
          src = ./.;
          CGO_ENABLED = 0;
          ldflags = [ "-s" "-w" "-X main.Version=v${version}" ];
          meta = with lib; {
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
            license = licenses.bsd0;
            maintainers = with maintainers; [ bdd ];
          };
        };

        enumer = buildGo121Module rec {
          pname = "enumer";
          version = "1.5.8";
          src = fetchFromGitHub {
            owner = "dmarkham";
            repo = "enumer";
            rev = "v${version}";
            sha256 = "sha256-+YTsXYWVmJ32V/Eptip3WAiqIYv+6nqbdph0K2XzLdc=";
          };
          vendorSha256 = "sha256-+dCitvPz2JUbybXVJxUOo1N6+SUPCSjlacL8bTSlb7w=";
          meta = with lib; {
            description = "A Go tool to auto generate methods for your enums";
            license = licenses.bsd2;
            maintainers = with maintainers; [ bdd ];
          };
        };
      };
    }
  );
}
