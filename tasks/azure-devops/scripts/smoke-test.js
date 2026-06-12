const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");

const root = path.resolve(__dirname, "..");
const handler = path.join(root, "dist", "index.js");

function writeExecutable(file, content) {
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, content, { mode: 0o755 });
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function runHandler(name, env) {
  const result = spawnSync(process.execPath, [handler], {
    cwd: root,
    env: {
      ...process.env,
      INPUT_BASE: "base-sha",
      INPUT_HEAD: "head-sha",
      INPUT_BLOCKON: "high",
      INPUT_FEEDBACK: "balanced",
      ...env
    },
    encoding: "utf8"
  });
  if (result.status !== 0) {
    throw new Error(`${name} failed with ${result.status}\nstdout:\n${result.stdout}\nstderr:\n${result.stderr}`);
  }
  if (result.stdout.includes("result=Failed") || result.stdout.includes("type=error")) {
    throw new Error(`${name} reported Azure task failure\nstdout:\n${result.stdout}\nstderr:\n${result.stderr}`);
  }
}

function read(file) {
  return fs.readFileSync(file, "utf8");
}

function makeFakeDiffPal(file, argvFile) {
  writeExecutable(file, `#!/usr/bin/env bash
set -euo pipefail
printf '%s\\n' "$@" > "${argvFile}"
`);
}

function makeFakeNpm(file, npmArgvFile, diffpalArgvFile) {
  writeExecutable(file, `#!/usr/bin/env node
const fs = require("node:fs");
const path = require("node:path");
const args = process.argv.slice(2);
fs.writeFileSync(${JSON.stringify(npmArgvFile)}, args.join("\\n"));
const prefixIndex = args.indexOf("--prefix");
if (prefixIndex === -1 || !args[prefixIndex + 1]) {
  process.exit(2);
}
const root = args[prefixIndex + 1];
const bin = path.join(root, "bin", "diffpal");
fs.mkdirSync(path.dirname(bin), { recursive: true });
fs.writeFileSync(bin, '#!/usr/bin/env bash\\nset -euo pipefail\\nprintf "%s\\\\n" "$@" > ${JSON.stringify(diffpalArgvFile)}\\n', { mode: 0o755 });
`);
}

function testDefaultInstall() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "diffpal-ado-install-"));
  const fakeBin = path.join(dir, "bin");
  const npmArgv = path.join(dir, "npm-argv");
  const diffpalArgv = path.join(dir, "diffpal-argv");
  const agentTemp = path.join(dir, "agent-temp");
  fs.mkdirSync(agentTemp, { recursive: true });
  makeFakeNpm(path.join(fakeBin, "npm"), npmArgv, diffpalArgv);

  runHandler("default install", {
    AGENT_TEMPDIRECTORY: agentTemp,
    INPUT_INSTALL: "true",
    INPUT_DIFFPALVERSION: "0.1.2",
    PATH: `${fakeBin}${path.delimiter}${process.env.PATH || ""}`
  });

  assert(read(npmArgv).includes("@diffpal/diffpal@0.1.2"), "default install did not request the configured package version");
  assert(read(diffpalArgv).includes("review\nado"), "default install did not run diffpal review ado");
}

function testCustomPathSkipsInstall() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "diffpal-ado-custom-"));
  const fakeBin = path.join(dir, "bin");
  const npmArgv = path.join(dir, "npm-argv");
  const diffpalArgv = path.join(dir, "diffpal-argv");
  const agentTemp = path.join(dir, "agent-temp");
  fs.mkdirSync(agentTemp, { recursive: true });
  makeFakeNpm(path.join(fakeBin, "npm"), npmArgv, diffpalArgv);
  const customDiffPal = path.join(dir, "custom-diffpal");
  makeFakeDiffPal(customDiffPal, diffpalArgv);

  runHandler("custom path", {
    INPUT_INSTALL: "true",
    INPUT_DIFFPALPATH: customDiffPal,
    AGENT_TEMPDIRECTORY: agentTemp,
    PATH: `${fakeBin}${path.delimiter}${process.env.PATH || ""}`
  });

  assert(!fs.existsSync(npmArgv), "custom diffpalPath should skip npm install");
  assert(read(diffpalArgv).includes("review\nado"), "custom path did not run diffpal review ado");
}

function testInstallDisabledUsesPath() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "diffpal-ado-path-"));
  const fakeBin = path.join(dir, "bin");
  const npmArgv = path.join(dir, "npm-argv");
  const diffpalArgv = path.join(dir, "diffpal-argv");
  const agentTemp = path.join(dir, "agent-temp");
  fs.mkdirSync(agentTemp, { recursive: true });
  makeFakeNpm(path.join(fakeBin, "npm"), npmArgv, diffpalArgv);
  makeFakeDiffPal(path.join(fakeBin, "diffpal"), diffpalArgv);

  runHandler("install disabled", {
    INPUT_INSTALL: "false",
    AGENT_TEMPDIRECTORY: agentTemp,
    PATH: `${fakeBin}${path.delimiter}${process.env.PATH || ""}`
  });

  assert(!fs.existsSync(npmArgv), "install=false should skip npm install");
  assert(read(diffpalArgv).includes("review\nado"), "install=false did not run diffpal from PATH");
}

testDefaultInstall();
testCustomPathSkipsInstall();
testInstallDisabledUsesPath();
console.log("Azure DevOps task smoke tests passed");
