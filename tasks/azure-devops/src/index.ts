import { spawn } from "node:child_process";
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

function spawnDiffPal(command: string, args: string[]): Promise<number> {
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

async function run(): Promise<void> {
  hydrateSystemAccessToken();

  const diffpalPath = input("diffpalPath") || "diffpal";
  const command = tl.which(diffpalPath, true);
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
  addOptional(args, "--mode", input("mode"));
  addOptional(args, "--language", input("language"));
  addOptional(args, "--review-checks", input("reviewChecks"));
  addOptional(args, "--out", input("out"));
  addOptional(args, "--repo", input("repo"));
  addOptional(args, "--review-id", input("reviewId"));
  addOptional(args, "--max-files", input("maxFiles"));
  addOptional(args, "--context-lines", input("contextLines"));
  addOptional(args, "--max-patch-chars", input("maxPatchChars"));
  addOptional(args, "--max-files-per-chunk", input("maxFilesPerChunk"));

  tl.debug(`Running ${command} ${args.join(" ")}`);
  const code = await spawnDiffPal(command, args);
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
