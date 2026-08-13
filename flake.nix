{
  description = "Repora deterministic repository management CLI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-26.05";
  };

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [
        "x86_64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      pkgsFor = system:
        import nixpkgs {
          inherit system;
          config.allowUnfreePredicate = pkg: nixpkgs.lib.getName pkg == "repora";
        };
      version = "0.1.0-dev";
      vendorHash = nixpkgs.lib.fakeHash;
      commit =
        if self ? shortRev then self.shortRev
        else if self ? dirtyShortRev then self.dirtyShortRev
        else "unknown";
    in
    {
      packages = forAllSystems (system:
        let
          pkgs = pkgsFor system;
          repora = pkgs.buildGoModule {
            pname = "repora";
            inherit version;
            src = self;
            inherit vendorHash;
            subPackages = [ "cmd/repoctl" ];
            doCheck = false;
            ldflags = [
              "-X main.version=${version}"
              "-X main.commit=${commit}"
            ];
            meta = {
              description = "Deterministic, policy-driven repository management system";
              homepage = "https://github.com/hackelia-micrantha/repora";
              license = {
                fullName = "Business Source License 1.1";
                spdxId = "BUSL-1.1";
                url = "https://mariadb.com/bsl11/";
                free = false;
              };
              mainProgram = "repoctl";
              platforms = supportedSystems;
            };
          };
        in
        {
          default = repora;
          inherit repora;
        });

      apps = forAllSystems (system:
        let
          repora = self.packages.${system}.repora;
        in
        {
          default = {
            type = "app";
            program = "${repora}/bin/repoctl";
            meta = {
              description = "Run the Repora repoctl CLI";
            };
          };
        });

      checks = forAllSystems (system:
        let
          pkgs = pkgsFor system;
          common = {
            pname = "repora-check";
            inherit version;
            src = self;
            inherit vendorHash;
            subPackages = [ ];
            doCheck = true;
            buildPhase = "true";
            nativeCheckInputs = [ pkgs.git ];
            installPhase = ''
              mkdir -p "$out"
              printf 'ok\n' > "$out/result"
            '';
          };
          mkGoCheck = name: command:
            pkgs.buildGoModule (common // {
              pname = "repora-${name}";
              checkPhase = ''
                runHook preCheck
                export HOME="$TMPDIR/home"
                mkdir -p "$HOME"
                export GIT_CONFIG_NOSYSTEM=1
                export GIT_TERMINAL_PROMPT=0
                ${command}
                runHook postCheck
              '';
            });
          unit = mkGoCheck "unit" "go test -race -count=1 -short ./...";
          integration = mkGoCheck "integration" "go test -race -count=1 ./internal/apply ./internal/managedartifact";
          staticAnalysis = mkGoCheck "static-analysis" "go vet ./...";
          smoke = pkgs.runCommand "repora-smoke" {
            nativeBuildInputs = [ self.packages.${system}.repora ];
          } ''
            mkdir -p "$out"
            repoctl --help > "$out/help.txt"
            repoctl --version > "$out/version.txt"
          '';
        in
        {
          inherit unit integration smoke;
          static-analysis = staticAnalysis;
          default = pkgs.linkFarm "repora-checks" [
            { name = "package"; path = self.packages.${system}.repora; }
            { name = "unit"; path = unit; }
            { name = "integration"; path = integration; }
            { name = "static-analysis"; path = staticAnalysis; }
            { name = "smoke"; path = smoke; }
          ];
        });

      devShells = forAllSystems (system:
        let
          pkgs = pkgsFor system;
        in
        {
          default = pkgs.mkShell {
            packages = [
              pkgs.go_1_25
              pkgs.git
              pkgs.gnumake
              pkgs.python3
              pkgs.nixfmt-rfc-style
            ];
          };
        });

      formatter = forAllSystems (system:
        let
          pkgs = pkgsFor system;
        in
        pkgs.nixfmt-rfc-style);
    };
}
