import { DotLottie } from '@lottiefiles/dotlottie-web';
import { createCanvas } from '@napi-rs/canvas';
import ffmpeg from "fluent-ffmpeg";
import pako from "pako";
import fs from "fs";
import path from "path";
import os from "os";
import { fileURLToPath } from 'url';
import { dirname } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Load WASM file
const wasmPath = path.resolve(__dirname, './node_modules/@lottiefiles/dotlottie-web/dist/dotlottie-player.wasm');
const wasmBase64 = fs.readFileSync(wasmPath).toString('base64');
const wasmDataUri = `data:application/wasm;base64,${wasmBase64}`;
DotLottie.setWasmUrl(wasmDataUri);

/**
 * Extract frames from TGS (Lottie JSON) and convert to animated WEBP
 * @param {Object|String} animation - Lottie JSON object or gzipped base64 string
 * @returns {Buffer} WEBP image buffer
 */
export async function tgsToWebp(animation) {
  const workDir = fs.mkdtempSync(path.join(os.tmpdir(), 'lottie-'));

  try {
    // If animation is base64 gzipped TGS, decompress it
    let lottieData = animation;
    if (typeof animation === 'string') {
      const buffer = Buffer.from(animation, 'base64');
      const decompressed = pako.ungzip(buffer, { to: 'string' });
      lottieData = JSON.parse(decompressed);
    }

    // Remove tgs field if present (not needed for rendering)
    if (lottieData.tgs) {
      delete lottieData.tgs;
    }

    const width = 100;
    const height = 100;
    const canvas = createCanvas(width, height);
    const ctx = canvas.getContext('2d');

    const framesDir = path.join(workDir, 'frames');
    fs.mkdirSync(framesDir, { recursive: true });

    const dotLottie = new DotLottie({
      canvas: canvas,
      data: JSON.stringify(lottieData),
      autoplay: true,
      useFrameInterpolation: false,
    });

    // Render all frames to PNG
    await new Promise((resolve, reject) => {
      let loadHandler, frameHandler, errorHandler, timeoutId;
      let frameIndex = 0;

      loadHandler = () => {
        console.log(`Rendering ${dotLottie.totalFrames} frames...`);
      };

      frameHandler = (event) => {
        try {
          // Save frame as PNG
          const framePath = path.join(framesDir, `frame_${String(frameIndex).padStart(5, '0')}.png`);
          const buffer = canvas.toBuffer('image/png');
          fs.writeFileSync(framePath, buffer);
          frameIndex++;

          if (event.currentFrame >= dotLottie.totalFrames) {
            // All frames rendered
            dotLottie.removeEventListener('load', loadHandler);
            dotLottie.removeEventListener('frame', frameHandler);
            dotLottie.removeEventListener('loadError', errorHandler);
            clearTimeout(timeoutId);

            dotLottie.stop();
            if (dotLottie.destroy) dotLottie.destroy();

            resolve();
          }
        } catch (err) {
          reject(err);
        }
      };

      errorHandler = (err) => {
        reject(new Error(`Failed to load Lottie: ${err}`));
      };

      dotLottie.addEventListener('load', loadHandler);
      dotLottie.addEventListener('frame', frameHandler);
      dotLottie.addEventListener('loadError', errorHandler);

      timeoutId = setTimeout(() => {
        reject(new Error('Timeout waiting for animation to complete'));
      }, 10000);
    });

    // Convert PNG frames to animated WEBP
    const webpPath = path.join(workDir, 'output.webp');
    await new Promise((resolve, reject) => {
      ffmpeg()
        .input(path.join(framesDir, 'frame_%05d.png'))
        .inputFPS(30)
        .outputOptions([
          '-vcodec libwebp',
          '-lossless 0',
          '-compression_level 6',
          '-q:v 90',
          '-loop 0',
          '-preset picture',
          '-an',
          '-vsync 0'
        ])
        .output(webpPath)
        .on('end', resolve)
        .on('error', reject)
        .run();
    });

    const webpBuffer = fs.readFileSync(webpPath);
    return webpBuffer;
  } finally {
    // Cleanup
    fs.rmSync(workDir, { recursive: true, force: true });
  }
}

/**
 * Convert WEBM video to animated WEBP
 * @param {String} base64Data - Base64 encoded WEBM video
 * @returns {Buffer} WEBP image buffer
 */
export async function webmToWebp(base64Data) {
  const workDir = fs.mkdtempSync(path.join(os.tmpdir(), 'lottie-'));

  try {
    const inputPath = path.join(workDir, 'input.webm');
    const outputPath = path.join(workDir, 'output.webp');

    // Write base64 data to file
    const buffer = Buffer.from(base64Data, 'base64');
    fs.writeFileSync(inputPath, buffer);

    // Convert WEBM to animated WEBP
    await new Promise((resolve, reject) => {
      ffmpeg(inputPath)
        .outputOptions([
          '-vcodec libwebp',
          '-lossless 0',
          '-compression_level 6',
          '-q:v 90',
          '-loop 0',
          '-preset picture',
          '-an',
          '-vsync 0',
          '-vf scale=100:100:force_original_aspect_ratio=decrease:flags=lanczos'
        ])
        .output(outputPath)
        .on('end', resolve)
        .on('error', reject)
        .run();
    });

    const webpBuffer = fs.readFileSync(outputPath);
    return webpBuffer;
  } finally {
    // Cleanup
    fs.rmSync(workDir, { recursive: true, force: true });
  }
}

/**
 * Resize and optimize WEBP (animated or static)
 * @param {String} base64Data - Base64 encoded WEBP image
 * @returns {Buffer} WEBP image buffer
 */
export async function webpToWebp(base64Data) {
  const workDir = fs.mkdtempSync(path.join(os.tmpdir(), 'lottie-'));

  try {
    const inputPath = path.join(workDir, 'input.webp');
    const outputPath = path.join(workDir, 'output.webp');

    // Write base64 data to file
    const buffer = Buffer.from(base64Data, 'base64');
    fs.writeFileSync(inputPath, buffer);

    // Resize and optimize WEBP
    await new Promise((resolve, reject) => {
      ffmpeg(inputPath)
        .outputOptions([
          '-vcodec libwebp',
          '-lossless 0',
          '-compression_level 6',
          '-q:v 90',
          '-loop 0',
          '-preset picture',
          '-an',
          '-vsync 0',
          '-vf scale=100:100:force_original_aspect_ratio=decrease:flags=lanczos'
        ])
        .output(outputPath)
        .on('end', resolve)
        .on('error', reject)
        .run();
    });

    const webpBuffer = fs.readFileSync(outputPath);
    return webpBuffer;
  } finally {
    // Cleanup
    fs.rmSync(workDir, { recursive: true, force: true });
  }
}
