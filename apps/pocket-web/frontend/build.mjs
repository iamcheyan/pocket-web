import { copyFile, mkdir } from 'node:fs/promises';

const result = Bun.spawnSync(['bun', 'build', 'src/main.tsx', '--outdir', 'dist', '--target', 'browser', '--format', 'iife'], { stdout: 'inherit', stderr: 'inherit' });
if (result.exitCode !== 0) process.exit(result.exitCode);
await mkdir('dist', { recursive: true });
await copyFile('index.html', 'dist/index.html');
