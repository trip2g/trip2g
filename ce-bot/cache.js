import fs from 'fs';
import path from 'path';

export class ThumbnailCache {
  constructor(cacheDir = '/tmp/lottiecache') {
    this.cacheDir = path.resolve(cacheDir);

    // Ensure cache directory exists
    if (!fs.existsSync(this.cacheDir)) {
      fs.mkdirSync(this.cacheDir, { recursive: true });
    }
  }

  /**
   * Validate that emojiId is a numeric string to prevent path traversal
   */
  _sanitize(emojiId) {
    const id = String(emojiId);
    if (!/^\d+$/.test(id)) throw new Error(`Invalid emojiId: ${emojiId}`);
    return id;
  }

  /**
   * Get path for emoji cache file
   */
  _getPath(emojiId) {
    return path.join(this.cacheDir, `${this._sanitize(emojiId)}.webp`);
  }

  /**
   * Get thumbnail from cache
   */
  get(emojiId) {
    const filePath = this._getPath(emojiId);

    if (!fs.existsSync(filePath)) {
      return null;
    }

    try {
      const webp_data = fs.readFileSync(filePath);
      return {
        webp_data,
        mime_type: 'image/webp'
      };
    } catch (err) {
      console.error(`Error reading cache file ${filePath}:`, err);
      return null;
    }
  }

  /**
   * Store thumbnail in cache
   */
  set(emojiId, webpData, mimeType = 'image/webp') {
    const filePath = this._getPath(emojiId);

    try {
      fs.writeFileSync(filePath, webpData);
    } catch (err) {
      console.error(`Error writing cache file ${filePath}:`, err);
      throw err;
    }
  }

  /**
   * Check if emoji is in cache
   */
  has(emojiId) {
    const filePath = this._getPath(emojiId);
    return fs.existsSync(filePath);
  }

  /**
   * Clean old thumbnails (older than maxAgeDays)
   */
  cleanup(maxAgeDays = 30) {
    const cutoff = Date.now() - (maxAgeDays * 24 * 60 * 60 * 1000);
    let count = 0;

    try {
      const files = fs.readdirSync(this.cacheDir);

      for (const file of files) {
        if (!file.endsWith('.webp')) continue;

        const filePath = path.join(this.cacheDir, file);
        const stats = fs.statSync(filePath);

        if (stats.mtimeMs < cutoff) {
          fs.unlinkSync(filePath);
          count++;
        }
      }
    } catch (err) {
      console.error('Error during cache cleanup:', err);
    }

    return count;
  }

  /**
   * Get cache stats
   */
  stats() {
    let count = 0;
    let sizeBytes = 0;

    try {
      const files = fs.readdirSync(this.cacheDir);

      for (const file of files) {
        if (!file.endsWith('.webp')) continue;

        const filePath = path.join(this.cacheDir, file);
        const stats = fs.statSync(filePath);

        count++;
        sizeBytes += stats.size;
      }
    } catch (err) {
      console.error('Error getting cache stats:', err);
    }

    return { count, sizeBytes };
  }

  close() {
    // No-op for file-based cache
  }
}
