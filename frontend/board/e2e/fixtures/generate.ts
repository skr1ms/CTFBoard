/**
 * Generate binary test fixture files.
 * Run once: bun e2e/fixtures/generate.ts
 */
import * as fs from 'fs'
import * as path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

// Minimal 1×1 transparent PNG
const PNG_HEX =
  '89504e470d0a1a0a0000000d49484452000000010000000108060000001f15c489' +
  '0000000a4944415478 9c620001000005000 10d0a2db4000000004945 4e44ae426082'
const pngBytes = Buffer.from(PNG_HEX.replace(/\s/g, ''), 'hex')
fs.writeFileSync(path.join(__dirname, 'avatar-static.png'), pngBytes)
console.log('wrote avatar-static.png')

// Minimal 2-frame animated GIF89a
const GIF_HEX =
  '47494638 39610100 01008000 00ffffff 00000021 ff0b4e45 54534341 504532 2e30' +
  '03010000 0021f904 000a0000 002c0000 00000100 01000002 024401 00' +
  '21f90400 14000000 002c0000 00000100 01000002 024c0100 3b'
const gifBytes = Buffer.from(GIF_HEX.replace(/\s/g, ''), 'hex')
fs.writeFileSync(path.join(__dirname, 'avatar-animated.gif'), gifBytes)
console.log('wrote avatar-animated.gif')
