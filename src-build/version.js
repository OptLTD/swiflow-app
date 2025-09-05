#!/usr/bin/env node

import { execSync } from 'child_process';
import fs from 'fs';
import path from 'path';

// Version types
const VERSION_TYPES = ['patch', 'minor', 'major'];

// File paths
const PACKAGE_JSON_PATH = path.resolve('package.json');
const TAURI_CONFIG_PATH = path.resolve('src-tauri/tauri.conf.json');
const CARGO_TOML_PATH = path.resolve('src-tauri/Cargo.toml');

/**
 * Parse command line arguments
 */
function parseArgs() {
  const args = process.argv.slice(2);
  
  if (args.includes('--help') || args.includes('-h')) {
    showHelp();
    process.exit(0);
  }
  
  const versionType = args[0] || 'patch';
  
  if (!VERSION_TYPES.includes(versionType)) {
    console.error(`❌ Invalid version type: ${versionType}`);
    console.error(`   Supported types: ${VERSION_TYPES.join(', ')}`);
    process.exit(1);
  }
  
  return { versionType };
}

/**
 * Show help information
 */
function showHelp() {
  console.log(`
📦 Version Management Tool

Usage:
  npm run build:version [type]
  npm run build:version:patch
  npm run build:version:minor
  npm run build:version:major

Version Types:
  patch    Increment patch version (x.y.Z)
  minor    Increment minor version (x.Y.0)
  major    Increment major version (X.0.0)

Options:
  --help, -h    Show this help message

Examples:
  npm run build:version patch
  npm run build:version:minor
`);
}

/**
 * Update package.json version using npm version command
 */
function updatePackageVersion(versionType) {
  console.log(`📦 Updating package.json version (${versionType})...`);
  
  try {
    const result = execSync(`npm version ${versionType} --no-git-tag-version`, { 
      encoding: 'utf8',
      cwd: process.cwd()
    });
    
    const newVersion = result.trim().replace('v', '');
    console.log(`✅ Updated package.json to version ${newVersion}`);
    return newVersion;
  } catch (error) {
    console.error('❌ Failed to update package.json version:', error.message);
    process.exit(1);
  }
}

/**
 * Update tauri.conf.json version
 */
function updateTauriConfig(version) {
  console.log('🔧 Updating tauri.conf.json version...');
  
  try {
    if (!fs.existsSync(TAURI_CONFIG_PATH)) {
      console.warn('⚠️  tauri.conf.json not found, skipping...');
      return;
    }
    
    const configContent = fs.readFileSync(TAURI_CONFIG_PATH, 'utf8');
    const config = JSON.parse(configContent);
    
    if (!config.version) {
      console.warn('⚠️  No version field found in tauri.conf.json');
      return;
    }
    
    config.version = version;
    
    fs.writeFileSync(TAURI_CONFIG_PATH, JSON.stringify(config, null, 2));
    console.log(`✅ Updated tauri.conf.json to version ${version}`);
  } catch (error) {
    console.error('❌ Failed to update tauri.conf.json:', error.message);
    process.exit(1);
  }
}

/**
 * Update Cargo.toml version
 */
function updateCargoToml(version) {
  console.log('🦀 Updating Cargo.toml version...');
  
  try {
    if (!fs.existsSync(CARGO_TOML_PATH)) {
      console.warn('⚠️  Cargo.toml not found, skipping...');
      return;
    }
    
    let cargoContent = fs.readFileSync(CARGO_TOML_PATH, 'utf8');
    
    // Replace version in [package] section
    cargoContent = cargoContent.replace(
      /^version\s*=\s*"[^"]*"/m,
      `version = "${version}"`
    );
    
    fs.writeFileSync(CARGO_TOML_PATH, cargoContent);
    console.log(`✅ Updated Cargo.toml to version ${version}`);
  } catch (error) {
    console.error('❌ Failed to update Cargo.toml:', error.message);
    process.exit(1);
  }
}

/**
 * Main function
 */
function main() {
  console.log('🚀 Starting version update process...\n');
  
  const { versionType } = parseArgs();
  
  // Update package.json version and get the new version
  const newVersion = updatePackageVersion(versionType);
  
  // Update other configuration files
  updateTauriConfig(newVersion);
  updateCargoToml(newVersion);
  
  console.log(`\n🎉 Successfully updated all files to version ${newVersion}`);
  console.log('\n📝 Files updated:');
  console.log('   - package.json');
  console.log('   - src-tauri/tauri.conf.json');
  console.log('   - src-tauri/Cargo.toml');
}

// Run the script
if (import.meta.url === `file://${process.argv[1]}`) {
  main();
}

export { updatePackageVersion, updateTauriConfig, updateCargoToml };