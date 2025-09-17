#!/usr/bin/env node

import { execSync } from 'child_process';
import { join, basename } from 'path';
import { 
  existsSync, mkdirSync,
  readFileSync, writeFileSync,
  rmSync, cpSync, chmodSync
} from 'fs';

// Show help information
function showHelp() {
  console.log(`
🚀 Cross-platform Build Script

Usage:
  node build.js [platform] [architecture] [options]

Platforms:
  mac     - Build for macOS
  win     - Build for Windows
  all     - Build for all platforms (default)

Architectures:
  For macOS: arm64, x86_64, all (default)
  For Windows: x86_64, all (default)

Options:
  --debug      - Enable debug mode (affects build configuration)
  --dry-run    - Run in dry-run mode (skip actual builds)
  --help, -h   - Show this help message

Version Management:
  Use separate version commands before building:
  npm run build:version:patch     # Increment patch version (x.y.Z)
  npm run build:version:minor     # Increment minor version (x.Y.0)
  npm run build:version:major     # Increment major version (X.0.0)

Examples:
  npm run build:version:patch     # Update version first
  node build.js                   # Build all platforms and architectures
  node build.js mac arm64         # Build macOS ARM64 only
  node build.js win x86_64 --dry-run # Dry-run Windows x86_64 build
`);
}

// Parse and validate command line arguments
function parseArgs() {
  const args = process.argv.slice(2);
  
  // Check for help flag
  if (args.includes('--help') || args.includes('-h')) {
    showHelp();
    process.exit(0);
  }
  
  // Check for flags
  const isDryRun = args.includes('--dry-run');
  const isDebug = args.includes('--debug');
  
  // Filter out flags to get positional arguments
  const positionalArgs = args.filter(arg => !arg.startsWith('--'));
  const [platform = 'all', arch = 'all'] = positionalArgs;
  
  // Validate platform
  const validPlatforms = ['mac', 'win', 'all'];
  if (!validPlatforms.includes(platform)) {
    console.error(`❌ Invalid platform '${platform}'. Use: ${validPlatforms.join(', ')}`);
    process.exit(1);
  }
  
  // Validate architecture for specific platforms
  const validateArchitecture = (targetPlatform, targetArch) => {
    if (targetArch === 'all') return true;
    
    // Define valid architectures inline to avoid initialization order issues
    const validArchitectures = {
      mac: ['all', 'arm64', 'x86_64'],
      win: ['all', 'x86_64']
    };
    
    const validArchs = validArchitectures[targetPlatform];
    if (validArchs && !validArchs.includes(targetArch)) {
      const archList = validArchs.filter(a => a !== 'all').join(', ');
      console.error(`❌ Invalid ${targetPlatform === 'mac' ? 'macOS' : 'Windows'} architecture '${targetArch}'. Use: all, ${archList}`);
      process.exit(1);
    }
  };
  
  // Validate architecture based on target platform(s)
  if (platform === 'mac' || platform === 'all') {
    validateArchitecture('mac', arch);
  }
  if (platform === 'win' || platform === 'all') {
    validateArchitecture('win', arch);
  }
  
  return { platform, arch, isDryRun, isDebug };
}

const { platform: targetPlatform, arch: targetArch, isDryRun, isDebug } = parseArgs();

// Build configuration constants
const BUILD_CONFIG = {
  // Core settings
  app: {
    name: 'main',
    expectedOutput: 'nice to meet you~',
    version: {
      file: 'package.json',
      epigraphInfo: ''
    }
  },
  
  // Build commands and paths
  build: {
    frontendCmd: 'yarn build',
    debugMode: isDebug,
    dryRun: isDryRun
  },
  
  // Directory structure
  paths: {
    source: 'src-core',
    mainFile: 'main.go',
    bin: 'bin',
    output: 'output',
    tauri: {
      src: 'src-tauri',
      config: 'src-tauri/tauri.conf.json',
      cargo: 'src-tauri/Cargo.toml'
    }
  }
};

// Platform architecture mappings
const PLATFORM_CONFIGS = {
  mac: {
    architectures: {
      arm64: { goarch: 'arm64', target: 'aarch64-apple-darwin' },
      x86_64: { goarch: 'amd64', target: 'x86_64-apple-darwin' }
    },
    buildTags: '!windows',
    ldflags: (version, epigraph) => {
      return `-X 'main.Version=${version}' -X 'main.Epigraph=${epigraph}' -s -w`
    } 
  },
  
  win: {
    architectures: {
      // x86: { goarch: '386', target: 'i686-pc-windows-msvc' },
      x86_64: { goarch: 'amd64', target: 'x86_64-pc-windows-msvc' }
    },
    buildTags: 'windows',
    ldflags: (version, epigraph) => {
      return `-X 'main.Version=${version}' -X 'main.Epigraph=${epigraph}' -s -w` // -H windowsgui
    }
  }
};

// Utility functions
function log(message, type = 'info') {
  const prefix = {
    info: '🚀',
    success: '✅',
    error: '❌',
    build: '🛠️',
    party: '🎉'
  };
  console.log(`${prefix[type] || '📝'} ${message}`);
}

function execCommand(command, options = {}) {
  const { cwd = process.cwd(), silent = false } = options;
  
  try {
    if (!silent) {
      log(`Executing: ${command}`);
    }
    
    // In dry-run mode, just log the command without executing
    if (BUILD_CONFIG.build.dryRun) {
      log(`[DRY-RUN] Would execute: ${command}`, 'info');
      return '';
    }
    
    const result = execSync(command, { 
      cwd, encoding: 'utf8',
      stdio: silent ? 'pipe' : 'inherit',
      env: process.env  // Pass environment variables including Apple API keys
    });
    return result;
  } catch (error) {
    if (BUILD_CONFIG.build.dryRun) {
      log(`[DRY-RUN] Command would fail: ${command}`, 'error');
      return '';
    }
    throw new Error(`Command failed: ${command}\n${error.message}`);
  }
}

// Apple API Keys loading function
function loadAppleApiKeys() {
  const appleKeysPath = join(process.env.HOME || '', '.apple_api_keys');
  
  if (!existsSync(appleKeysPath)) {
    log('Apple API keys file not found at ~/.apple_api_keys', 'error');
    return false;
  }
  
  try {
    // Load Apple API keys by sourcing the file
    const keysContent = readFileSync(appleKeysPath, 'utf8');
    const lines = keysContent.split('\n');
    
    lines.forEach(line => {
      const trimmedLine = line.trim();
      if (trimmedLine && !trimmedLine.startsWith('#') && trimmedLine.includes('=')) {
        // Handle both 'export KEY=value' and 'KEY=value' formats
        let processedLine = trimmedLine;
        if (processedLine.startsWith('export ')) {
          processedLine = processedLine.substring(7); // Remove 'export ' prefix
        }
        
        const [key, ...valueParts] = processedLine.split('=');
        const value = valueParts.join('=').replace(/^["']|["']$/g, ''); // Remove quotes
        process.env[key.trim()] = value.trim();
      }
    });
    
    log('Apple API keys loaded successfully');
    return true;
  } catch (error) {
    log(`Failed to load Apple API keys: ${error.message}`, 'error');
    return false;
  }
}

// Get current version from package.json
function getCurrentVersion() {
  try {
    const packageJson = JSON.parse(readFileSync(BUILD_CONFIG.app.version.file, 'utf8'));
    return packageJson.version;
  } catch (error) {
    log(`Failed to read version from ${BUILD_CONFIG.app.version.file}: ${error.message}`, 'error');
    throw error;
  }
}

function cleanOldBuilds() {
  log('Cleaning old builds...');
  
  const dirsToClean = [
    { path: BUILD_CONFIG.paths.bin, recreate: true },
    { path: BUILD_CONFIG.paths.output, recreate: true },
    { path: join(BUILD_CONFIG.paths.tauri.src, 'target'), recreate: false }
  ];
  
  dirsToClean.forEach(({ path, recreate }) => {
    try {
      if (existsSync(path)) {
        rmSync(path, { recursive: true, force: true });
        if (recreate) {
          mkdirSync(path, { recursive: true });
        }
      }
    } catch (error) {
      // Silently ignore cleanup errors
    }
  });
}

function buildFrontend() {
  log('Building frontend...');
  execCommand(BUILD_CONFIG.build.frontendCmd);
}

function ensureDirectories() {
  const requiredDirs = [BUILD_CONFIG.paths.bin, BUILD_CONFIG.paths.output];
  
  requiredDirs.forEach(dir => {
    if (!existsSync(dir)) {
      mkdirSync(dir, { recursive: true });
    }
  });
}

// Platform-specific build functions
function buildGoBinary(platform, arch, version) {
  const platformConfig = PLATFORM_CONFIGS[platform];
  const archConfig = platformConfig.architectures[arch];
  
  if (!archConfig) {
    throw new Error(`Unsupported architecture '${arch}' for platform '${platform}'`);
  }
  
  const { goarch, target } = archConfig;
  const ldflags = platformConfig.ldflags(version, BUILD_CONFIG.app.version.epigraphInfo);
  
  const outputFile = platform === 'mac' 
    ? join(BUILD_CONFIG.paths.bin, `${BUILD_CONFIG.app.name}-${target}`)
    : `${BUILD_CONFIG.paths.bin}/${BUILD_CONFIG.app.name}-${target}.exe`;
  
  const envVars = platform === 'win' 
    ? `env GOOS=windows GOARCH=${goarch} CGO_ENABLED=1`
    : `env GOARCH=${goarch} CGO_ENABLED=1`;
  
  const buildCmd = `${envVars} go build -tags='${platformConfig.buildTags}' -ldflags "${ldflags}" -o "../${outputFile}"`;
  
  log(`Building Go binary for ${platform}/${arch} (${target})...`, 'build');
  execCommand(buildCmd, { cwd: BUILD_CONFIG.paths.source });
  
  return outputFile;
}

function buildMacOSPlatform(arch, version) {
  return new Promise((resolve, reject) => {
    const archConfig = PLATFORM_CONFIGS.mac.architectures[arch];
    const { target } = archConfig;
    
    log(`Building macOS ${arch} (${target})...`, 'build');

    try {
      // Build Go binary
      const outputFile = buildGoBinary('mac', arch, version);
      log('Go binary build completed', 'success');

      // Build Tauri package
      const debugOption = BUILD_CONFIG.build.debugMode ? '--debug' : '';
      const tauriCmd = `yarn tauri build ${debugOption} --target "${target}" --config '{"build": {"beforeBuildCommand": "echo skip"}}'`;
      execCommand(tauriCmd);

      // Copy DMG files to output directory
      copyMacOSOutputs(target, version);

      log(`macOS ${arch} build completed!`, 'success');
      resolve();
    } catch (error) {
      log(`macOS ${arch} build failed: ${error.message}`, 'error');
      reject(error);
    }
  });
}

function copyMacOSOutputs(target, version) {
  const dmgSourcePath = join(BUILD_CONFIG.paths.tauri.src, 'target', target, 'release', 'bundle', 'dmg');
  
  if (!existsSync(dmgSourcePath)) {
    log('No DMG files found to copy', 'info');
    return;
  }
  
  try {
    const dmgFiles = execSync(`ls "${dmgSourcePath}"/*.dmg 2>/dev/null || true`, { encoding: 'utf8' }).trim();
    
    if (dmgFiles) {
      dmgFiles.split('\n').forEach(dmgFile => {
        if (dmgFile.trim()) {
          const targetPath = join(BUILD_CONFIG.paths.output, basename(dmgFile.trim()));
          cpSync(dmgFile.trim(), targetPath);
          log(`Copied DMG: ${basename(dmgFile.trim())}`);
        }
      });
      
      // Handle epigraph version renaming if configured
      handleEpigraphRenaming(version);
    }
  } catch (error) {
    log(`Failed to copy DMG files: ${error.message}`, 'error');
  }
}

function handleEpigraphRenaming(version) {
  const epigraphInfo = BUILD_CONFIG.app.version.epigraphInfo;
  if (!epigraphInfo) return;
  
  try {
    const epigraphName = epigraphInfo.split('|')[0];
    const files = execSync(`ls "${BUILD_CONFIG.paths.output}"/*.dmg 2>/dev/null || true`, { encoding: 'utf8' }).trim();
    
    if (files) {
      files.split('\n').forEach(file => {
        if (file.includes(version)) {
          const newName = file.replace(version, epigraphName);
          execSync(`mv "${file}" "${newName}"`);
          log(`Renamed: ${basename(file)} -> ${basename(newName)}`);
        }
      });
    }
  } catch (error) {
    log(`Failed to rename files: ${error.message}`, 'error');
  }
}

function buildForWindows(arch, version) {
  const archName = arch === 'amd64' ? 'x86_64' : '';
  log(`Building Windows ${archName}...`, 'build');
  
  try {
    // Build Go binary using unified function
    const outputPath = buildGoBinary('win', archName, version);
    log(`Windows ${archName} build completed`, 'success');
    
    // Copy to SWIFLOW_WIN_DIR if environment variable is set
    copyToWindowsEnvironment(outputPath);
    
    return true;
  } catch (error) {
    log(`Windows ${archName} build failed: ${error.message}`, 'error');
    return false;
  }
}

function copyToWindowsEnvironment(outputPath) {
  const swiflowWinDir = process.env.SWIFLOW_WIN_DIR;
  
  if (!swiflowWinDir || !existsSync(join(swiflowWinDir, BUILD_CONFIG.paths.tauri.config))) {
    return;
  }
  
  log(`Copying files to Windows environment: ${swiflowWinDir}`);
  
  try {
    // Copy binary file
    const targetBinDir = join(swiflowWinDir, 'bin');
    if (!existsSync(targetBinDir)) {
      mkdirSync(targetBinDir, { recursive: true });
    }
    
    cpSync(outputPath, join(targetBinDir, basename(outputPath)));
    log(`Copied binary: ${basename(outputPath)}`);
    
    // Copy dist directory if it exists
    if (existsSync('dist')) {
      const targetDistDir = join(swiflowWinDir, 'dist');
      if (existsSync(targetDistDir)) {
        rmSync(targetDistDir, { recursive: true, force: true });
      }
      cpSync('dist', targetDistDir, { recursive: true });
      log(`Copied dist directory to ${targetDistDir}`);
    }
  } catch (error) {
    log(`Failed to copy to Windows environment: ${error.message}`, 'error');
  }
}

// Binary verification functions
function verifyBinary(binaryPath) {
  log(`Verifying binary: ${basename(binaryPath)}`);
  
  try {
    chmodSync(binaryPath, '755');
    // Use shell command to capture both stdout and stderr
    const output = execSync(`${binaryPath} 2>&1 | head -10`, { 
      encoding: 'utf8', shell: true
    });
    
    if (output.includes(BUILD_CONFIG.app.expectedOutput)) {
      log('Binary verification passed', 'success');
      return true;
    } else {
      log(`Binary verification failed - expected '${BUILD_CONFIG.app.expectedOutput}' in output`, 'error');
      log(`Actual output: ${output.substring(0, 200)}...`, 'debug');
      return false;
    }
  } catch (error) {
    log(`Binary verification failed: ${error.message}`, 'error');
    return false;
  }
}

function getCurrentPlatformBinary() {
  const platform = process.platform;
  const arch = process.arch;
  
  if (platform === 'darwin') {
    const macArch = arch === 'arm64' ? 'arm64' : 'x86_64';
    const target = PLATFORM_CONFIGS.mac.architectures[macArch].target;
    return join(BUILD_CONFIG.paths.bin, `${BUILD_CONFIG.app.name}-${target}`);
  } else if (platform === 'win32') {
    const winArch = arch === 'x86_64' ? 'x86_64' : '';
    const target = PLATFORM_CONFIGS.win.architectures[winArch].target;
    return `${BUILD_CONFIG.paths.bin}/${BUILD_CONFIG.app.name}-${target}.exe`;
  }
  
  return null;
}

function getArchitecturesToBuild(platform, targetArch) {
  if (targetArch === 'all') {
    return Object.keys(PLATFORM_CONFIGS[platform].architectures);
  }
  return [targetArch];
}

// Main build function
async function main() {
  try {
    log(`Starting cross-platform build process...`);
    log(`Target platform: ${targetPlatform}, Target architecture: ${targetArch}`);
    
    if (BUILD_CONFIG.build.dryRun) {
      log(`🔍 Running in DRY-RUN mode - no actual builds will be performed`, 'info');
    }

    // Get current version
    const currentVersion = getCurrentVersion();
    log(`Building version: ${currentVersion}`);
    
    // Note: Version management is now handled separately via 'npm run build:version' commands

    cleanOldBuilds();
    buildFrontend();
    ensureDirectories();

    let allSuccess = true;

    // Build macOS if requested
    if (targetPlatform === 'mac' || targetPlatform === 'all') {
      allSuccess = await buildMacOSTargets(currentVersion, targetArch) && allSuccess;
    }

    // Build Windows if requested
    if (targetPlatform === 'win' || targetPlatform === 'all') {
      allSuccess = buildWindowsTargets(currentVersion, targetArch) && allSuccess;
    }

    // Final verification and summary
    await finalizeBuild(allSuccess);
    
    process.exit(allSuccess ? 0 : 1);
  } catch (error) {
    log(`Build process failed: ${error.message}`, 'error');
    process.exit(1);
  }
}

async function buildMacOSTargets(version, targetArch) {
  log('Building macOS platforms...', 'build');
  
  // Load Apple API keys for macOS builds (required for code signing and notarization)
  const epigraphInfo = BUILD_CONFIG.app.version.epigraphInfo;
  const isDebugMode = BUILD_CONFIG.build.debugMode;
  
  // Load Apple API keys unless in debug mode without epigraph info
  // This matches the shell script logic: load keys when epigraph exists OR when not in debug mode
  if (!isDebugMode || epigraphInfo) {
    if (!loadAppleApiKeys()) {
      log('❌ Failed to load Apple API keys. This may affect code signing and notarization.', 'error');
      // Continue with build but warn user
    }
  }
  
  const architectures = getArchitecturesToBuild('mac', targetArch);
  
  try {
    // Build all macOS architectures in parallel
    const buildPromises = architectures.map(arch => {
      return buildMacOSPlatform(arch, version);
    });
    await Promise.all(buildPromises);
    
    const archText = targetArch === 'all' ? 'all architectures' : targetArch;
    log(`macOS build for ${archText} completed!`, 'success');
    return true;
  } catch (error) {
    log(`macOS build failed: ${error.message}`, 'error');
    return false;
  }
}

function buildWindowsTargets(version, targetArch) {
  log('Building Windows platforms...', 'build');
  
  const architectures = getArchitecturesToBuild('win', targetArch);
  let allSuccess = true;
  
  // Build Windows architectures sequentially
  for (const arch of architectures) {
    const goArch = PLATFORM_CONFIGS.win.architectures[arch].goarch;
    const success = buildForWindows(goArch, version);
    if (!success) {
      allSuccess = false;
    }
  }
  
  const archText = targetArch === 'all' ? 'all architectures' : targetArch;
  log(`Windows build for ${archText} ${allSuccess ? 'completed!' : 'failed!'}`, allSuccess ? 'success' : 'error');
  
  return allSuccess;
}

async function finalizeBuild(allSuccess) {
  if (allSuccess) {
    log('All builds completed successfully!', 'party');
    
    // List output files
    listOutputFiles();
    
    // Verify current platform binary if available
    const currentBinary = getCurrentPlatformBinary();
    if (currentBinary && existsSync(currentBinary)) {
      log('Running final verification...');
      if (verifyBinary(currentBinary)) {
        log('All builds and verification completed successfully!', 'party');
      } else {
        log('Build completed but verification failed!', 'error');
        return false;
      }
    }
  } else {
    log('Some builds failed!', 'error');
  }
  
  return allSuccess;
}

function listOutputFiles() {
  try {
    log('Output files:');
    
    if (existsSync(BUILD_CONFIG.paths.bin)) {
      const binFiles = execCommand(`ls -lh ${BUILD_CONFIG.paths.bin}/`, { silent: true });
      console.log(binFiles);
    }
    
    if (existsSync(BUILD_CONFIG.paths.output)) {
      const outputFiles = execCommand(`ls -lh ${BUILD_CONFIG.paths.output}/`, { silent: true });
      console.log(outputFiles);
    }
  } catch (error) {
    // Silently ignore listing errors
  }
}

// Run the main function if this script is executed directly
if (import.meta.url === `file://${process.argv[1]}`) {
  main();
}

export default main;