const fs = require("node:fs");
const path = require("node:path");

const root = path.resolve(__dirname, "..");
const source = path.join(root, "dist", "index.js");
const taskDirs = ["DiffPalReviewV1", "DiffPalReviewDevV1"];
const lock = JSON.parse(fs.readFileSync(path.join(root, "package-lock.json"), "utf8"));

function collectRuntimeDependencies() {
  const rootPackage = lock.packages[""];
  const pending = Object.keys(rootPackage.dependencies || {});
  const seen = new Set();

  while (pending.length > 0) {
    const name = pending.pop();
    if (!name || seen.has(name)) {
      continue;
    }
    seen.add(name);

    const packagePath = `node_modules/${name}`;
    const packageInfo = lock.packages[packagePath];
    if (!packageInfo) {
      continue;
    }
    for (const dependency of Object.keys(packageInfo.dependencies || {})) {
      pending.push(dependency);
    }
    for (const dependency of Object.keys(packageInfo.optionalDependencies || {})) {
      pending.push(dependency);
    }
  }

  return [...seen].sort();
}

function copyRuntimeDependencies(taskRoot, dependencies) {
  const targetNodeModules = path.join(taskRoot, "node_modules");
  fs.rmSync(targetNodeModules, { recursive: true, force: true });

  for (const dependency of dependencies) {
    const sourceDir = path.join(root, "node_modules", ...dependency.split("/"));
    if (!fs.existsSync(sourceDir)) {
      continue;
    }
    const targetDir = path.join(targetNodeModules, ...dependency.split("/"));
    fs.mkdirSync(path.dirname(targetDir), { recursive: true });
    fs.rmSync(targetDir, { recursive: true, force: true });
    fs.cpSync(sourceDir, targetDir, { recursive: true });
  }
}

const runtimeDependencies = collectRuntimeDependencies();

for (const taskDir of taskDirs) {
  const taskRoot = path.join(root, taskDir);
  const targetDir = path.join(taskRoot, "dist");
  fs.mkdirSync(targetDir, { recursive: true });
  fs.copyFileSync(source, path.join(targetDir, "index.js"));
  copyRuntimeDependencies(taskRoot, runtimeDependencies);
}
