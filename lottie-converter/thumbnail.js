import { DotLottie } from '@lottiefiles/dotlottie-web';
import { createCanvas } from '@napi-rs/canvas';
import GIFEncoder from 'gif-encoder';
import ffmpeg from "fluent-ffmpeg";
import pako from "pako";
import fs from "fs";
import path from "path";
import crypto from "crypto";
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
 * Extract first frame from TGS (Lottie JSON) and convert to GIF
 * @param {Object|String} animation - Lottie JSON object or gzipped base64 string
 * @returns {Buffer} GIF image buffer
 */
export async function tgsToGif(animation) {
  const jobId = crypto.randomUUID();
  const workDir = `/tmp/${jobId}`;
  fs.mkdirSync(workDir, { recursive: true });

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

    const gifPath = path.join(workDir, 'output.gif');
    const gif = new GIFEncoder(width, height);

    const dotLottie = new DotLottie({
      canvas: canvas,
      data: JSON.stringify(lottieData),
      autoplay: true,
      useFrameInterpolation: false,
    });

    // Wait for all frames to render
    await new Promise((resolve, reject) => {
      let loadHandler, frameHandler, errorHandler, timeoutId;
      let file;

      loadHandler = () => {
        file = fs.createWriteStream(gifPath);

        gif.pipe(file);
        gif.setRepeat(0); // Loop forever
        gif.setFrameRate(30);
        gif.setTransparent(0x00000000);
        gif.setQuality(1); // Best quality (1-20, lower is better)
        gif.writeHeader();
      };

      frameHandler = (event) => {
        try {
          // Get frame data and add to GIF
          const frameData = ctx.getImageData(0, 0, width, height).data;

          if (event.currentFrame >= dotLottie.totalFrames) {
            // All frames captured, finish GIF
            gif.finish();

            // Wait for file to finish writing
            file.on('finish', () => {
              // Cleanup all event listeners
              dotLottie.removeEventListener('load', loadHandler);
              dotLottie.removeEventListener('frame', frameHandler);
              dotLottie.removeEventListener('loadError', errorHandler);
              clearTimeout(timeoutId);

              // Stop and destroy DotLottie
              dotLottie.stop();
              if (dotLottie.destroy) dotLottie.destroy();

              resolve();
            });
          } else {
            gif.addFrame(frameData);
            gif._read(1);
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

      // Timeout after 10 seconds for full animation
      timeoutId = setTimeout(() => {
        reject(new Error('Timeout waiting for animation to complete'));
      }, 10000);
    });

    const gifBuffer = fs.readFileSync(gifPath);
    return gifBuffer;
  } finally {
    // Cleanup
    fs.rmSync(workDir, { recursive: true, force: true });
  }
}

/**
 * Extract first frame from WEBM video and convert to GIF
 * @param {String} base64Data - Base64 encoded WEBM video
 * @returns {Buffer} GIF image buffer
 */
export async function webmToGif(base64Data) {
  const jobId = crypto.randomUUID();
  const workDir = `/tmp/${jobId}`;
  fs.mkdirSync(workDir, { recursive: true });

  try {
    const inputPath = path.join(workDir, 'input.webm');
    const outputPath = path.join(workDir, 'output.gif');

    // Write base64 data to file
    const buffer = Buffer.from(base64Data, 'base64');
    fs.writeFileSync(inputPath, buffer);

    // Extract first frame and convert to GIF
    await new Promise((resolve, reject) => {
      ffmpeg(inputPath)
        .outputOptions([
          '-vframes 1',
          '-vf scale=100:100:force_original_aspect_ratio=decrease,pad=100:100:(ow-iw)/2:(oh-ih)/2',
        ])
        .output(outputPath)
        .on('end', resolve)
        .on('error', reject)
        .run();
    });

    const gifBuffer = fs.readFileSync(outputPath);
    return gifBuffer;
  } finally {
    // Cleanup
    fs.rmSync(workDir, { recursive: true, force: true });
  }
}

/**
 * Convert WEBP to GIF (first frame)
 * @param {String} base64Data - Base64 encoded WEBP image
 * @returns {Buffer} GIF image buffer
 */
export async function webpToGif(base64Data) {
  const jobId = crypto.randomUUID();
  const workDir = `/tmp/${jobId}`;
  fs.mkdirSync(workDir, { recursive: true });

  try {
    const inputPath = path.join(workDir, 'input.webp');
    const outputPath = path.join(workDir, 'output.gif');

    // Write base64 data to file
    const buffer = Buffer.from(base64Data, 'base64');
    fs.writeFileSync(inputPath, buffer);

    // Convert WEBP to GIF
    await new Promise((resolve, reject) => {
      ffmpeg(inputPath)
        .outputOptions([
          '-vf scale=100:100:force_original_aspect_ratio=decrease,pad=100:100:(ow-iw)/2:(oh-ih)/2',
        ])
        .output(outputPath)
        .on('end', resolve)
        .on('error', reject)
        .run();
    });

    const gifBuffer = fs.readFileSync(outputPath);
    return gifBuffer;
  } finally {
    // Cleanup
    fs.rmSync(workDir, { recursive: true, force: true });
  }
}
