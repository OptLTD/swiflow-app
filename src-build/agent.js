#!/usr/bin/env node

import { execSync } from 'child_process';
import { existsSync, readFileSync } from 'fs';
import { join, resolve, basename } from 'path';
import { createWriteStream } from 'fs';
import archiver from 'archiver';
import FormData from 'form-data';
import fetch from 'node-fetch';
import { nanoid } from 'nanoid';

/**
 * Agent Start Command
 * Usage: npm run agent:start <directory>
 * 
 * This command performs two main steps:
 * 1. Compress the .agent directory from specified directory into name.agent file and import via Import API
 * 2. Read task.md as task input and call Start API to begin conversation
 */

// Configuration
const AGENT_DIR = '.agent';
const TASK_FILE = 'task.md';
const SERVER_URL = 'http://localhost:11235';

// Parse command line arguments
const args = process.argv.slice(2);
if (args.length === 0) {
  console.log('Usage: npm run agent:start <directory>');
  console.log('   or: yarn agent:start <directory>');
  console.log('');
  console.log('Description:');
  console.log('  This command performs two main steps:');
  console.log('  1. Compress the .agent directory from specified directory into name.agent file and import via Import API');
  console.log('  2. Read task.md as task input and call Start API to begin conversation');
  console.log('');
  console.log('Arguments:');
  console.log('  <directory>    Path to directory containing .agent folder and task.md file');
  console.log('');
  console.log('Environment Variables:');
  console.log('  SERVER_URL    Server URL (default: http://localhost:11235)');
  console.log('');
  console.log('Examples:');
  console.log('  npm run agent:start ./my-project');
  console.log('  yarn agent:start ./my-project');
  console.log('  npm run agent:start /path/to/agent/directory');
  console.log('  yarn agent:start /path/to/agent/directory');
  process.exit(0);
}

const targetDirectory = resolve(args[0]);
const serverUrl = process.env.SERVER_URL || SERVER_URL;

// Utility functions
function log(message, type = 'info') {
  const prefix = {
    info: '🚀',
    success: '✅',
    error: '❌',
    warning: '⚠️'
  };
  console.log(`${prefix[type] || '📝'} ${message}`);
}

function validateDirectory(dir) {
  if (!existsSync(dir)) {
    throw new Error(`Directory does not exist: ${dir}`);
  }
  
  const agentDir = join(dir, AGENT_DIR);
  if (!existsSync(agentDir)) {
    throw new Error(`${AGENT_DIR} directory not found in: ${dir}`);
  }
  
  const taskFile = join(dir, TASK_FILE);
  if (!existsSync(taskFile)) {
    throw new Error(`${TASK_FILE} file not found in: ${dir}`);
  }
  
  return { agentDir, taskFile };
}

async function compressAgentDirectory(agentDir, outputPath) {
  return new Promise((resolve, reject) => {
    const output = createWriteStream(outputPath);
    const archive = archiver('zip', {
      zlib: { level: 9 } // Maximum compression
    });

    output.on('close', () => {
      log(`Agent directory compressed: ${archive.pointer()} bytes`, 'success');
      resolve();
    });

    archive.on('error', (err) => {
      reject(err);
    });

    archive.pipe(output);
    
    // Add all files from .agent directory
    archive.directory(agentDir, false);
    
    archive.finalize();
  });
}

async function importAgent(agentFilePath) {
  try {
    const form = new FormData();
    form.append('files', createReadStream(agentFilePath));
    const response = await fetch(`${serverUrl}/api/import`, {
      method: 'POST', body: form,
      headers: form.getHeaders()
    });
    
    if (!response.ok) {
      throw new Error(`Import API failed: ${response.status} ${response.statusText}`);
    }
    
    const result = await response.json();
    log(`Agent imported successfully: ${JSON.stringify(result.imported)}`, 'success');
    return result;
  } catch (error) {
    throw new Error(`Failed to import agent: ${error.message}`);
  } finally {
    // Cleanup: Remove temporary agent file
    try {
      execSync(`rm -f "${agentFilePath}"`);
    } catch (cleanupError) {
      log(`Warning: Failed to cleanup temporary file: ${agentFilePath}`, 'warning');
    }
  }
}

async function startConversation(leaderId, taskContent) {
  try {
    const requestBody = {
      content: taskContent.trim(),
      startNew: "yes",
      taskUUID: nanoid(12),
      workerId: leaderId,
      homePath: targetDirectory,
    };
    
    const response = await fetch(`${serverUrl}/api/start`, {
      method: 'POST', headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(requestBody)
    });
    
    if (!response.ok) {
      throw new Error(`Start API failed: ${response.status} ${response.statusText}`);
    }
    
    const result = await response.json();
    log('Conversation started successfully', 'success');
    return result;
  } catch (error) {
    throw new Error(`Failed to start conversation: ${error.message}`);
  }
}

// Main execution
async function main() {
  try {
    log(`Starting agent from directory: ${targetDirectory}`);
    
    // Step 1: Validate directory structure
    log('Validating directory structure...');
    const { agentDir, taskFile } = validateDirectory(targetDirectory);
    
    // Step 2: Compress .agent directory
    log('Compressing .agent directory...');
    const dirName = basename(targetDirectory);
    const agentFileName = `${dirName}.agent`;
    const agentFilePath = join(process.cwd(), agentFileName);
    await compressAgentDirectory(agentDir, agentFilePath);
    
    // Step 3: Import agent via API
    log('Importing agent...');
    const importResult = await importAgent(agentFilePath);
    
    // Step 4: Read task.md content
    log('Reading task content...');
    const taskContent = readFileSync(taskFile, 'utf-8');
    
    if (!taskContent.trim()) {
      throw new Error('task.md file is empty');
    }
    
    // Step 5: Start conversation
    log('Starting conversation...');
    await startConversation(importResult.leaderId, taskContent);
    log('Agent start process completed successfully! 🎉', 'success');
  } catch (error) {
    log(`Error: ${error.message}`, 'error');
    process.exit(1);
  }
}

// Add missing imports at the top
import { createReadStream } from 'fs';

// Run main function
main().catch((error) => {
  log(`Unexpected error: ${error.message}`, 'error');
  process.exit(1);
});