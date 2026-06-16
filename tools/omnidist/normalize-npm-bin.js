#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const npmRoot = path.join(process.cwd(), '.omnidist', 'default', 'npm', '@diffpal');
const metaPackage = '@diffpal/diffpal';
const binName = 'diffpal';
const platformSuffix = /-(darwin|linux|win32)-/;

function readJSON(file) {
  return JSON.parse(fs.readFileSync(file, 'utf8'));
}

function writeJSON(file, value) {
  fs.writeFileSync(file, JSON.stringify(value, null, 2) + '\n');
}

if (!fs.existsSync(npmRoot)) {
  throw new Error(`staged npm package root does not exist: ${npmRoot}`);
}

const packages = fs.readdirSync(npmRoot, { withFileTypes: true })
  .filter((entry) => entry.isDirectory())
  .map((entry) => path.join(npmRoot, entry.name, 'package.json'))
  .filter((file) => fs.existsSync(file));

const metaFile = packages.find((file) => readJSON(file).name === metaPackage);
if (!metaFile) {
  throw new Error(`missing staged meta package ${metaPackage}`);
}

const meta = readJSON(metaFile);
if (!meta.bin || meta.bin[binName] !== 'diffpal.js') {
  throw new Error(`${metaPackage} must expose bin.${binName}=diffpal.js`);
}

let changed = 0;
for (const file of packages) {
  const pkg = readJSON(file);
  if (pkg.name === metaPackage) {
    continue;
  }
  if (!pkg.name || !pkg.name.startsWith('@diffpal/diffpal-') || !platformSuffix.test(pkg.name)) {
    continue;
  }
  if (!pkg.bin) {
    continue;
  }
  delete pkg.bin;
  writeJSON(file, pkg);
  changed++;
}

if (changed === 0) {
  throw new Error('no platform package bin fields were removed');
}

console.log(`normalized npm bin ownership: ${metaPackage} owns ${binName}; removed platform bin fields from ${changed} package(s)`);
