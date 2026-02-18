{
  description = "EdgeClaw — Autonomous IoT monitoring fleet powered by PicoClaw + Qwen";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11";
    nixpkgs-unstable.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

    flake-utils.url = "github:numtide/flake-utils";

    microvm = {
      url = "github:astro/microvm.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    gomod2nix = {
      url = "github:nix-community/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, nixpkgs-unstable, flake-utils, microvm, gomod2nix }:
    let
      # Supported build hosts
      supportedSystems = [ "x86_64-linux" "aarch64-linux" ];

      # Cross-compilation target triples
      crossTargets = {
        arm64 = {
          GOOS = "linux";
          GOARCH = "arm64";
          suffix = "arm64";
          nixSystem = "aarch64-linux";
        };
        riscv64 = {
          GOOS = "linux";
          GOARCH = "riscv64";
          suffix = "riscv64";
          nixSystem = "riscv64-linux";
        };
      };

      version = self.shortRev or self.dirtyShortRev or "dev";

      # Build the Go binary for a given pkgs set and optional cross-compilation env.
      mkEdgeclaw = { pkgs, GOOS ? "linux", GOARCH ? null, suffix ? "" }:
        pkgs.buildGoModule {
          pname = "edgeclaw";
          inherit version;
          src = self;

          vendorHash = null; # uses vendor/ or Go module proxy

          subPackages = [ "cmd/edgeclaw" ];

          ldflags = [
            "-s" "-w"
            "-trimpath"
            "-X main.version=${version}"
          ];

          CGO_ENABLED = "0";
          inherit GOOS;
        } // pkgs.lib.optionalAttrs (GOARCH != null) {
          inherit GOARCH;
        };

    in
    flake-utils.lib.eachSystem supportedSystems (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ gomod2nix.overlays.default ];
        };
        pkgs-unstable = import nixpkgs-unstable { inherit system; };

        # Native build
        edgeclaw = mkEdgeclaw { inherit pkgs; };

        # Cross-compiled builds
        edgeclaw-arm64 = mkEdgeclaw {
          inherit pkgs;
          GOOS = "linux";
          GOARCH = "arm64";
          suffix = "arm64";
        };
        edgeclaw-riscv64 = mkEdgeclaw {
          inherit pkgs;
          GOOS = "linux";
          GOARCH = "riscv64";
          suffix = "riscv64";
        };

        # -------------------------------------------------------------------
        #  Helper scripts
        # -------------------------------------------------------------------
        ec-dev = pkgs.writeShellScriptBin "ec-dev" ''
          set -euo pipefail
          echo "EdgeClaw dev stack — starting TimescaleDB, Mosquitto, Ollama..."
          ${pkgs.docker-compose}/bin/docker-compose \
            -f ${self}/deploy/docker-compose.yml up -d "$@"
          echo ""
          echo "Services:"
          echo "  TimescaleDB  → localhost:5432"
          echo "  Mosquitto    → localhost:1883 (ws: 9001)"
          echo "  Ollama       → localhost:11434"
        '';

        ec-dev-down = pkgs.writeShellScriptBin "ec-dev-down" ''
          set -euo pipefail
          ${pkgs.docker-compose}/bin/docker-compose \
            -f ${self}/deploy/docker-compose.yml down "$@"
          echo "EdgeClaw dev stack stopped."
        '';

        ec-seed = pkgs.writeShellScriptBin "ec-seed" ''
          set -euo pipefail
          export PGHOST="''${PGHOST:-localhost}"
          export PGPORT="''${PGPORT:-5432}"
          export PGUSER="''${PGUSER:-edgeclaw}"
          export PGPASSWORD="''${PGPASSWORD:-changeme}"
          export PGDATABASE="''${PGDATABASE:-edgeclaw}"

          echo "EdgeClaw seed — applying schema + demo data..."
          ${pkgs.postgresql_16}/bin/psql -v ON_ERROR_STOP=1 \
            -f ${self}/deploy/schema.sql
          ${pkgs.postgresql_16}/bin/psql -v ON_ERROR_STOP=1 \
            -f ${self}/scripts/seed.sql
          echo "Seed complete."
        '';

        ec-lint = pkgs.writeShellScriptBin "ec-lint" ''
          set -euo pipefail
          echo "Running go vet..."
          ${pkgs.go_1_22}/bin/go vet ./...
          echo "Running staticcheck..."
          ${pkgs.go-tools}/bin/staticcheck ./...
          echo "Lint passed."
        '';

        ec-test = pkgs.writeShellScriptBin "ec-test" ''
          set -euo pipefail
          echo "Running tests with race detector..."
          ${pkgs.go_1_22}/bin/go test -race -count=1 -v ./... "$@"
        '';

        ec-build-all = pkgs.writeShellScriptBin "ec-build-all" ''
          set -euo pipefail
          mkdir -p bin
          echo "Building edgeclaw (native)..."
          ${pkgs.go_1_22}/bin/go build -trimpath \
            -ldflags "-s -w -X main.version=${version}" \
            -o bin/edgeclaw ./cmd/edgeclaw

          echo "Building edgeclaw (arm64)..."
          GOOS=linux GOARCH=arm64 ${pkgs.go_1_22}/bin/go build -trimpath \
            -ldflags "-s -w -X main.version=${version}" \
            -o bin/edgeclaw-arm64 ./cmd/edgeclaw

          echo "Building edgeclaw (riscv64)..."
          GOOS=linux GOARCH=riscv64 ${pkgs.go_1_22}/bin/go build -trimpath \
            -ldflags "-s -w -X main.version=${version}" \
            -o bin/edgeclaw-riscv64 ./cmd/edgeclaw

          echo ""
          ls -lh bin/edgeclaw*
          echo "All targets built."
        '';

        ec-workspace-init = pkgs.writeShellScriptBin "ec-workspace-init" ''
          set -euo pipefail
          WORKSPACE="''${HOME}/.picoclaw/workspace"
          echo "Installing PicoClaw workspace templates → $WORKSPACE"
          mkdir -p "$WORKSPACE/skills/iot-monitor"
          cp ${self}/workspace/HEARTBEAT.md  "$WORKSPACE/"
          cp ${self}/workspace/IDENTITY.md   "$WORKSPACE/"
          cp ${self}/workspace/SOUL.md       "$WORKSPACE/"
          cp ${self}/workspace/AGENTS.md     "$WORKSPACE/"
          cp ${self}/workspace/skills/iot-monitor/SKILL.md \
             "$WORKSPACE/skills/iot-monitor/"
          echo "Workspace initialized."
        '';

        ec-mqtt-setup = pkgs.writeShellScriptBin "ec-mqtt-setup" ''
          set -euo pipefail
          ${pkgs.bash}/bin/bash ${self}/scripts/setup-mqtt-users.sh "$@"
        '';

        helperScripts = [
          ec-dev
          ec-dev-down
          ec-seed
          ec-lint
          ec-test
          ec-build-all
          ec-workspace-init
          ec-mqtt-setup
        ];

      in {

        # ===================================================================
        #  Packages
        # ===================================================================
        packages = {
          default = edgeclaw;
          edgeclaw = edgeclaw;
          edgeclaw-arm64 = edgeclaw-arm64;
          edgeclaw-riscv64 = edgeclaw-riscv64;

          # OCI image matching the Dockerfile output
          docker = pkgs.dockerTools.buildLayeredImage {
            name = "edgeclaw";
            tag = version;
            contents = [
              edgeclaw
              pkgs.cacert
              pkgs.tzdata
            ];
            config = {
              Entrypoint = [ "${edgeclaw}/bin/edgeclaw" ];
              Cmd = [ "-config" "/etc/edgeclaw/config.json" ];
              Env = [
                "SSL_CERT_FILE=${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt"
                "TZ=UTC"
              ];
            };
          };
        };

        # ===================================================================
        #  Development shell
        # ===================================================================
        devShells.default = pkgs.mkShell {
          name = "edgeclaw-dev";

          packages = with pkgs; [
            # Go toolchain
            go_1_22
            gopls
            gotools
            go-tools          # staticcheck
            delve             # debugger
            golangci-lint

            # Database
            postgresql_16     # psql client

            # MQTT
            mosquitto         # mosquitto_pub / mosquitto_sub

            # Infrastructure
            docker-compose
            docker-client

            # Networking
            tailscale
            curl
            jq

            # Nix tooling
            nil               # Nix LSP
            nixpkgs-fmt

            # General
            gnumake
            git
          ] ++ helperScripts;

          shellHook = ''
            echo ""
            echo "  ╔══════════════════════════════════════════════════════╗"
            echo "  ║           EdgeClaw Development Environment          ║"
            echo "  ╠══════════════════════════════════════════════════════╣"
            echo "  ║                                                      ║"
            echo "  ║  Build        │ make build / ec-build-all            ║"
            echo "  ║  Test         │ make test  / ec-test                 ║"
            echo "  ║  Lint         │ ec-lint                              ║"
            echo "  ║  Dev stack    │ ec-dev / ec-dev-down                 ║"
            echo "  ║  Seed DB      │ ec-seed                              ║"
            echo "  ║  Workspace    │ ec-workspace-init                    ║"
            echo "  ║  MQTT users   │ ec-mqtt-setup                        ║"
            echo "  ║                                                      ║"
            echo "  ║  MicroVM      │ nix run .#microvm                    ║"
            echo "  ║  Docker img   │ nix build .#docker                   ║"
            echo "  ║  Cross ARM64  │ nix build .#edgeclaw-arm64           ║"
            echo "  ║  Cross RV64   │ nix build .#edgeclaw-riscv64         ║"
            echo "  ║                                                      ║"
            echo "  ╚══════════════════════════════════════════════════════╝"
            echo ""
          '';

          CGO_ENABLED = "0";
        };

        # ===================================================================
        #  Checks (nix flake check)
        # ===================================================================
        checks = {
          build = edgeclaw;
          lint = pkgs.runCommand "edgeclaw-lint" {
            nativeBuildInputs = [ pkgs.go_1_22 ];
            src = self;
          } ''
            cd $src
            export HOME=$(mktemp -d)
            export GOFLAGS="-mod=mod"
            go vet ./...
            touch $out
          '';
        };

        # ===================================================================
        #  App (nix run)
        # ===================================================================
        apps.default = {
          type = "app";
          program = "${edgeclaw}/bin/edgeclaw";
        };
      }
    ) // {

      # =====================================================================
      #  NixOS module — for integration into NixOS / microvm configurations
      # =====================================================================
      nixosModules.edgeclaw = { config, lib, pkgs, ... }:
        let
          cfg = config.services.edgeclaw;
        in {
          options.services.edgeclaw = {
            enable = lib.mkEnableOption "EdgeClaw IoT monitoring agent";

            configFile = lib.mkOption {
              type = lib.types.path;
              default = "/etc/edgeclaw/config.json";
              description = "Path to the PicoClaw/EdgeClaw configuration file.";
            };

            nodeId = lib.mkOption {
              type = lib.types.str;
              default = "edge-bot-01";
              description = "Unique identifier for this fleet node.";
            };

            package = lib.mkOption {
              type = lib.types.package;
              default = self.packages.${pkgs.system}.edgeclaw;
              description = "The edgeclaw package to use.";
            };

            workspaceDir = lib.mkOption {
              type = lib.types.path;
              default = "/var/lib/edgeclaw/workspace";
              description = "Directory for PicoClaw workspace templates.";
            };
          };

          config = lib.mkIf cfg.enable {
            systemd.services.edgeclaw = {
              description = "EdgeClaw IoT monitoring agent";
              after = [ "network-online.target" "postgresql.service" ];
              wants = [ "network-online.target" ];
              wantedBy = [ "multi-user.target" ];

              serviceConfig = {
                Type = "simple";
                ExecStart = "${cfg.package}/bin/edgeclaw -config ${cfg.configFile}";
                Restart = "on-failure";
                RestartSec = 10;

                # Hardening
                DynamicUser = true;
                StateDirectory = "edgeclaw";
                ProtectSystem = "strict";
                ProtectHome = true;
                NoNewPrivileges = true;
                PrivateTmp = true;
                ReadOnlyPaths = [ cfg.configFile cfg.workspaceDir ];
              };

              environment = {
                EDGECLAW_NODE_ID = cfg.nodeId;
              };
            };
          };
        };

      # =====================================================================
      #  MicroVM — lightweight Linux VM for integration testing
      # =====================================================================
      nixosConfigurations.microvm = nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";
        modules = [
          microvm.nixosModules.microvm
          self.nixosModules.edgeclaw
          ({ config, pkgs, lib, ... }: {
            # ---------------------------------------------------------------
            #  MicroVM hypervisor configuration
            # ---------------------------------------------------------------
            microvm = {
              hypervisor = "qemu";
              vcpu = 2;
              mem = 1024; # 1 GB — enough for Go binary + TimescaleDB

              interfaces = [{
                type = "user";
                id = "ec-net";
              }];

              shares = [{
                tag = "ro-store";
                source = "/nix/store";
                mountPoint = "/nix/.ro-store";
                proto = "virtiofs";
              }];

              volumes = [{
                image = "edgeclaw-data.img";
                mountPoint = "/var/lib/edgeclaw";
                size = 512; # MB
              }];
            };

            # ---------------------------------------------------------------
            #  System
            # ---------------------------------------------------------------
            networking.hostName = "edgeclaw-vm";
            time.timeZone = "UTC";

            # ---------------------------------------------------------------
            #  Services — PostgreSQL + TimescaleDB
            # ---------------------------------------------------------------
            services.postgresql = {
              enable = true;
              package = pkgs.postgresql_16;
              ensureDatabases = [ "edgeclaw" ];
              ensureUsers = [{
                name = "edgeclaw";
                ensureDBOwnership = true;
              }];
              authentication = ''
                local all all trust
                host all all 127.0.0.1/32 trust
              '';
            };

            # ---------------------------------------------------------------
            #  Services — Mosquitto MQTT broker
            # ---------------------------------------------------------------
            services.mosquitto = {
              enable = true;
              listeners = [{
                port = 1883;
                acl = [ "topic readwrite edgeclaw/#" ];
                omitPasswordAuth = true;
                settings.allow_anonymous = true;
              }];
            };

            # ---------------------------------------------------------------
            #  EdgeClaw agent
            # ---------------------------------------------------------------
            services.edgeclaw = {
              enable = true;
              nodeId = "microvm-test-01";
              configFile = pkgs.writeText "edgeclaw-config.json" (builtins.toJSON {
                agents.defaults = {
                  workspace = "/var/lib/edgeclaw/workspace";
                  model = "qwen3-0.6b";
                  max_tokens = 4096;
                  temperature = 0.3;
                  max_tool_iterations = 10;
                  restrict_to_workspace = true;
                };
                providers.qwen_brain = {
                  api_base = "http://localhost:11434/v1";
                  model = "qwen3-0.6b";
                  timeout_seconds = 60;
                };
                heartbeat = {
                  enabled = false; # disabled for testing
                  interval = 5;
                };
                edgeclaw = {
                  node_id = "microvm-test-01";
                  database = {
                    host = "localhost";
                    port = 5432;
                    user = "edgeclaw";
                    password = "";
                    dbname = "edgeclaw";
                    sslmode = "disable";
                    max_conns = 5;
                  };
                  mqtt = {
                    broker_url = "tcp://localhost:1883";
                    topic_prefix = "edgeclaw/fleet";
                    username = "";
                    password = "";
                  };
                };
              });
            };

            # ---------------------------------------------------------------
            #  Schema initialisation (run once after PostgreSQL starts)
            # ---------------------------------------------------------------
            systemd.services.edgeclaw-schema = {
              description = "Apply EdgeClaw database schema";
              after = [ "postgresql.service" ];
              requires = [ "postgresql.service" ];
              before = [ "edgeclaw.service" ];
              wantedBy = [ "multi-user.target" ];
              serviceConfig = {
                Type = "oneshot";
                RemainAfterExit = true;
                User = "edgeclaw";
                ExecStart = "${pkgs.postgresql_16}/bin/psql -d edgeclaw -f ${self}/deploy/schema.sql";
              };
            };

            # ---------------------------------------------------------------
            #  Packages available in the VM
            # ---------------------------------------------------------------
            environment.systemPackages = with pkgs; [
              self.packages.x86_64-linux.edgeclaw
              postgresql_16
              mosquitto
              curl
              jq
              htop
            ];

            # ---------------------------------------------------------------
            #  Users
            # ---------------------------------------------------------------
            users.users.root.password = "edgeclaw";
            services.getty.autologinUser = "root";

            system.stateVersion = "24.11";
          })
        ];
      };

      # =====================================================================
      #  Overlay — makes edgeclaw available as pkgs.edgeclaw
      # =====================================================================
      overlays.default = final: prev: {
        edgeclaw = self.packages.${prev.system}.edgeclaw;
      };
    };
}
