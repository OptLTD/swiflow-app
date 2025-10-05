#!/usr/bin/env node

/**
 * Docker build script for Swiflow application
 * Handles Docker image building with proper platform targeting
 */

import { execSync } from 'child_process';
import path from 'path';
import { fileURLToPath } from 'url';

// ES module compatibility
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Configuration
const CONFIG = {
    platform: 'linux/amd64',
    dockerfile: './Dockerfile',
    imageName: 'optltd/swiflow:latest',
    buildContext: './'
};

/**
 * Parse command line arguments
 */
function parseArgs() {
    const args = process.argv.slice(2);
    const options = {
        help: false,
        dryRun: false,
        push: false,
        tag: CONFIG.imageName
    };

    for (let i = 0; i < args.length; i++) {
        const arg = args[i];
        switch (arg) {
            case '--help':
            case '-h':
                options.help = true;
                break;
            case '--dry-run':
                options.dryRun = true;
                break;
            case '--push':
                options.push = true;
                break;
            case '--tag':
            case '-t':
                if (i + 1 < args.length) {
                    options.tag = args[++i];
                } else {
                    console.error('Error: --tag requires a value');
                    process.exit(1);
                }
                break;
            default:
                console.error(`Error: Unknown argument '${arg}'`);
                process.exit(1);
        }
    }

    return options;
}

/**
 * Display help information
 */
function showHelp() {
    console.log(`
Swiflow Docker Build Script

Usage: node src-build/docker.js [options]

Options:
  --help, -h     Show this help message
  --dry-run      Show commands without executing them
  --push         Push image to registry after building
  --tag, -t      Specify image tag (default: ${CONFIG.imageName})

Examples:
  node src-build/docker.js                    # Build image
  node src-build/docker.js --push             # Build and push image
  node src-build/docker.js --tag myapp:v1.0   # Build with custom tag
  node src-build/docker.js --dry-run          # Preview commands

NPM Scripts:
  npm run build:docker                        # Build image
  npm run build:docker -- --push              # Build and push image
`);
}

/**
 * Execute command with proper error handling
 */
function executeCommand(command, dryRun = false) {
    console.log(`> ${command}`);
    
    if (dryRun) {
        console.log('  [DRY RUN] Command would be executed');
        return;
    }

    try {
        execSync(command, { 
            stdio: 'inherit',
            cwd: process.cwd()
        });
    } catch (error) {
        console.error(`Error executing command: ${command}`);
        console.error(error.message);
        process.exit(1);
    }
}

/**
 * Pre-build steps: build frontend and copy assets to backend initial html
 */
function preBuildAssets(options) {
    const { dryRun } = options;

    console.log('\n=== Pre-Build Frontend Assets ===\n');
    // Build frontend assets
    executeCommand('yarn build', dryRun);

    // Ensure target directory exists
    executeCommand('mkdir -p src-core/initial/html', dryRun);

    // Clean up existing files in target directory
    executeCommand('rm -rf src-core/initial/html/*', dryRun);

    // Copy built assets
    executeCommand('cp -R dist/* src-core/initial/html/', dryRun);
}

/**
 * Build Docker image
 */
function buildDockerImage(options) {
    const { tag, dryRun, push } = options;
    
    // Run pre-build steps before Docker build
    preBuildAssets(options);
    
    console.log('\n=== Docker Build Process ===\n');
    console.log(`Platform: ${CONFIG.platform}`);
    console.log(`Dockerfile: ${CONFIG.dockerfile}`);
    console.log(`Image Tag: ${tag}`);
    console.log(`Build Context: ${CONFIG.buildContext}`);
    console.log('');

    // Build command
    const buildCommand = [
        'docker buildx build',
        `--platform ${CONFIG.platform}`,
        `-f ${CONFIG.dockerfile}`,
        `-t ${tag}`,
        CONFIG.buildContext
    ].join(' ');

    executeCommand(buildCommand, dryRun);

    if (push) {
        console.log('\n=== Pushing Image ===\n');
        const pushCommand = `docker push ${tag}`;
        executeCommand(pushCommand, dryRun);
    }

    if (!dryRun) {
        console.log('\n=== Build Complete ===\n');
        console.log(`Image built successfully: ${tag}`);
        if (push) {
            console.log('Image pushed to registry');
        }
        console.log('\nUsage:');
        console.log(`  docker run -d --name swiflow --network host ${tag}`);
        console.log(`  docker run -d --name swiflow -p 11235:11235 ${tag}`);
    }
}

/**
 * Main function
 */
function main() {
    const options = parseArgs();

    if (options.help) {
        showHelp();
        return;
    }

    buildDockerImage(options);
}

// Execute if run directly
if (import.meta.url === `file://${process.argv[1]}`) {
    main();
}

export { buildDockerImage, parseArgs };