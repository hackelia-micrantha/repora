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
      # Pin the patched official Go source without changing the locked Nixpkgs
      # snapshot. The original Nixpkgs compiler is vulnerable (Go 1.26.5).
      patchedGoFor = pkgs:
        pkgs.go_1_26.overrideAttrs (_: {
          version = "1.26.10";
          src = pkgs.fetchurl {
            url = "https://go.dev/dl/go1.26.10.src.tar.gz";
            hash = "sha256-/WPog7ovK/ypPBZGsx6+9j4rHk0rMXyXidl2po2B828=";
          };
        });
      patchedGoModuleFor = pkgs:
        pkgs.callPackage (nixpkgs + "/pkgs/build-support/go/module.nix") {
          go = patchedGoFor pkgs;
        };
      version = "0.3.0";
      vendorHash = "sha256-g+yaVIx4jxpAQ/+WrGKxhVeliYx7nLQe/zsGpxV4Fn4=";
      commit =
        if self ? shortRev then self.shortRev
        else if self ? dirtyShortRev then self.dirtyShortRev
        else "unknown";
    in
    {
      packages = forAllSystems (system:
        let
          pkgs = pkgsFor system;
          repora = (patchedGoModuleFor pkgs) {
            pname = "repora";
            inherit version;
            src = self;
            inherit vendorHash;
            subPackages = [ "cmd/repoctl" ];
            doCheck = false;
            ldflags = [
              "-X main.version=v${version}"
              "-X main.commit=${commit}"
            ];
            postInstall = ''
              install -d \
                "$out/share/man/man1" \
                "$out/share/repora/examples" \
                "$out/share/repora/schemas"
              install -m 0444 man/repoctl.1 "$out/share/man/man1/repoctl.1"
              install -m 0444 examples/repora.yaml "$out/share/repora/examples/repora.yaml"
              for schema in schemas/*.schema.json; do
                install -m 0444 "$schema" "$out/share/repora/schemas/"
              done
            '';
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
            nativeCheckInputs = [
              pkgs.git
              pkgs.gnumake
              pkgs.python3
              pkgs.go-tools
            ];
            installPhase = ''
              mkdir -p "$out"
              printf 'ok\n' > "$out/result"
            '';
          };
          mkGoCheck = name: command:
            (patchedGoModuleFor pkgs) (common // {
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
          format = mkGoCheck "format" "make format-check";
          unit = mkGoCheck "unit" "make test";
          integration = mkGoCheck "integration" "make integration";
          contract = mkGoCheck "contract" "make contract-test";
          e2e = mkGoCheck "e2e" "make e2e";
          staticAnalysis = mkGoCheck "static-analysis" "make static-analysis STATICCHECK=staticcheck";
          smoke = pkgs.runCommand "repora-smoke" {
            nativeBuildInputs = [ self.packages.${system}.repora (patchedGoFor pkgs) ];
          } ''
            mkdir -p "$out"
            repoctl --help > "$out/help.txt"
            repoctl --version > "$out/version.txt"
            go version -m "$(command -v repoctl)" > "$out/compiler.txt"
            grep -F "go1.26.10" "$out/compiler.txt"
            test -f "${self.packages.${system}.repora}/share/man/man1/repoctl.1.gz"
            test -f "${self.packages.${system}.repora}/share/repora/examples/repora.yaml"
            test -n "$(find "${self.packages.${system}.repora}/share/repora/schemas" -type f -name '*.schema.json' -print -quit)"
          '';
        in
        {
          inherit format unit integration contract e2e smoke;
          static-analysis = staticAnalysis;
          default = pkgs.linkFarm "repora-checks" [
            { name = "package"; path = self.packages.${system}.repora; }
            { name = "format"; path = format; }
            { name = "unit"; path = unit; }
            { name = "integration"; path = integration; }
            { name = "contract"; path = contract; }
            { name = "e2e"; path = e2e; }
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
              (patchedGoFor pkgs)
              pkgs.git
              pkgs.gnumake
              pkgs.python3
              pkgs.go-tools
              pkgs.nixfmt
            ];
          };
        });

      formatter = forAllSystems (system:
        let
          pkgs = pkgsFor system;
        in
        pkgs.nixfmt);
    };
}
