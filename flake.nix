{
  description = "runitor";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }: flake-utils.lib.eachDefaultSystem (system:
    let
      pkgs = import nixpkgs { inherit system; };
    in {
      devShells = {
        default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # build
            go
            self.packages.${system}.enumer

            # release
            gh  # make a release an github and upload artifacts
            git # mkrel: git tag, git push
            curl # verify, dlrel, build
            coreutils #  sha256sum: sign & verify
            openssh # ssh-keygen: sign & verify
          ];
        };
      };

      packages = with pkgs; {
        default = buildGoModule rec {
          pname = "runitor";
          revDate = builtins.substring 0 8 (self.lastModifiedDate or "19700101");
          version = "${revDate}-${self.shortRev or "dirty"}";
          vendorSha256 = null;
          src = ./.;
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

        enumer = buildGoModule rec {
          pname = "enumer";
          version = "1.5.7";
          src = fetchFromGitHub {
            owner = "dmarkham";
            repo = "enumer";
            rev = "v${version}";
            sha256 = "2fVWrrWOiCtg7I3Lul2PgQ2u/qDEDioPSB61Tp0rfEo=";
          };
          vendorSha256 = "BmFv0ytRnjaB7z7Gb+38Fw2ObagnaFMnMhlejhaGxsk=";
          meta = with lib; {
            description = "A Go tool to auto generate methods for your enums";
            license =  licenses.bsd2;
            maintainers = with maintainers; [ bdd ];
          };
        };

      };
    }
  );
}
