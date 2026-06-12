import { spawn } from "node:child_process";
import * as fs from "node:fs";
import * as path from "node:path";
import * as tl from "azure-pipelines-task-lib/task";

function input(name: string): string {
  return (tl.getInput(name, false) ?? "").trim();
}

function requireValue(name: string, value: string): string {
  if (!value) {
    throw new Error(`Required value is empty: ${name}`);
  }
  return value;
}

function firstEnv(names: string[]): string {
  for (const name of names) {
    const value = process.env[name];
    if (value && value.trim() !== "") {
      return value.trim();
    }
  }
  return "";
}

function hydrateSystemAccessToken(): void {
  if (process.env.SYSTEM_ACCESSTOKEN) {
    return;
  }
  const token = tl.getVariable("System.AccessToken");
  if (token) {
    process.env.SYSTEM_ACCESSTOKEN = token;
  }
}

function addOptional(args: string[], flag: string, value: string): void {
  if (value) {
    args.push(flag, value);
  }
}

function boolInput(name: string, defaultValue: boolean): boolean {
  const value = input(name).toLowerCase();
  if (!value) {
    return defaultValue;
  }
  return ["1", "true", "yes", "y", "on"].includes(value);
}

function resolveBase(inputBase: string): string {
  return inputBase || firstEnv([
    "SYSTEM_PULLREQUEST_TARGETCOMMITID",
    "SYSTEM_PULLREQUEST_TARGETBRANCH"
  ]);
}

function resolveHead(inputHead: string): string {
  return inputHead || firstEnv([
    "SYSTEM_PULLREQUEST_SOURCECOMMITID",
    "BUILD_SOURCEVERSION",
    "SYSTEM_PULLREQUEST_SOURCEBRANCH"
  ]);
}

function spawnCommand(command: string, args: string[]): Promise<number> {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, {
      env: process.env,
      shell: false,
      stdio: "inherit"
    });

    child.on("error", reject);
    child.on("close", (code) => resolve(code ?? 1));
  });
}

async function installDiffPal(version: string): Promise<string> {
  const npm = tl.which("npm", true);
  const tempDir = tl.getVariable("Agent.TempDirectory") || process.env.AGENT_TEMPDIRECTORY || process.env.RUNNER_TEMP || process.cwd();
  const installRoot = path.join(tempDir, "diffpal-task");
  fs.mkdirSync(installRoot, { recursive: true });

  const packageSpec = `@diffpal/diffpal@${version || "latest"}`;
  tl.debug(`Installing ${packageSpec} into ${installRoot}`);
  const code = await spawnCommand(npm, [
    "install",
    "--global",
    "--prefix",
    installRoot,
    packageSpec,
    "--omit=dev",
    "--no-audit",
    "--no-fund"
  ]);
  if (code !== 0) {
    throw new Error(`npm install ${packageSpec} exited with code ${code}`);
  }

  const candidates = process.platform === "win32"
    ? [
        path.join(installRoot, "diffpal.cmd"),
        path.join(installRoot, "diffpal"),
        path.join(installRoot, "bin", "diffpal.cmd")
      ]
    : [
        path.join(installRoot, "bin", "diffpal"),
        path.join(installRoot, "diffpal")
      ];

  const diffpal = candidates.find((candidate) => fs.existsSync(candidate));
  if (!diffpal) {
    throw new Error(`installed diffpal binary was not found in ${installRoot}`);
  }
  tl.debug(`Installed DiffPal binary: ${diffpal}`);
  return diffpal;
}

async function resolveDiffPalCommand(): Promise<string> {
  const diffpalPath = input("diffpalPath") || "diffpal";
  if (diffpalPath !== "diffpal") {
    return tl.which(diffpalPath, true);
  }
  if (!boolInput("install", true)) {
    return tl.which(diffpalPath, true);
  }
  return installDiffPal(input("diffpalVersion") || "latest");
}

async function run(): Promise<void> {
  hydrateSystemAccessToken();

  const command = await resolveDiffPalCommand();
  const base = requireValue("base or System.PullRequest.TargetCommitId", resolveBase(input("base")));
  const head = requireValue("head or System.PullRequest.SourceCommitId", resolveHead(input("head")));
  const blockOn = input("blockOn") || "high";
  const gate = tl.getBoolInput("gate", false);

  const args: string[] = [];
  addOptional(args, "--config-dir", input("configDir"));
  addOptional(args, "--profile", input("profile"));
  args.push("review", "ado", "--base", base, "--head", head, "--block-on", blockOn);

  if (gate) {
    args.push("--gate");
  }
  const mode = input("mode");
  addOptional(args, "--mode", mode);
  if (!mode) {
    addOptional(args, "--feedback", input("feedback") || "balanced");
  }
  addOptional(args, "--language", input("language"));
  addOptional(args, "--review-checks", input("reviewChecks"));
  addOptional(args, "--instructions", input("instructions"));
  addOptional(args, "--instructions-file", input("instructionsFile"));
  addOptional(args, "--out", input("out"));
  addOptional(args, "--repo", input("repo"));
  addOptional(args, "--review-id", input("reviewId"));
  addOptional(args, "--max-files", input("maxFiles"));
  addOptional(args, "--context-lines", input("contextLines"));
  addOptional(args, "--max-patch-chars", input("maxPatchChars"));
  addOptional(args, "--max-files-per-chunk", input("maxFilesPerChunk"));

  tl.debug(`Running ${command} ${args.join(" ")}`);
  const code = await spawnCommand(command, args);
  if (code !== 0) {
    process.exitCode = code;
    tl.setResult(tl.TaskResult.Failed, `diffpal exited with code ${code}`);
    return;
  }
  tl.setResult(tl.TaskResult.Succeeded, "DiffPal review completed");
}

run().catch((error: unknown) => {
  const message = error instanceof Error ? error.message : String(error);
  process.exitCode = 1;
  tl.setResult(tl.TaskResult.Failed, message);
});
