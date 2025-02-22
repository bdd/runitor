{ self, pkgs, ... }:

pkgs.buildGo124Module rec {
  pname = "runitor";
  revDate = builtins.substring 0 8 (self.lastModifiedDate or "19700101");
  version = "${revDate}-${self.shortRev or "dirty"}";
  vendorHash = "sha256-SYYAAtuWt/mTmZPBilYxf2uZ6OcgeTnobYiye47i8mI=";
  src = ./.;
  env.CGO_ENABLED = 0;
  ldflags = [
    "-s"
    "-w"
    "-X main.Version=v${version}"
  ];
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
}
