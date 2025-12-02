import express from 'express';
import { Telegraf } from 'telegraf';
import { ThumbnailCache } from './cache.js';
import { tgsToWebp, webmToWebp, webpToWebp } from './thumbnail.js';
import fetch from 'node-fetch';

const app = express();
const cache = new ThumbnailCache();

// Telegram Bot
const TELEGRAM_BOT_TOKEN = process.env.TELEGRAM_BOT_TOKEN;
const SERVER_URL = process.env.SERVER_URL || 'http://localhost:3000';

if (!TELEGRAM_BOT_TOKEN) {
  console.error('TELEGRAM_BOT_TOKEN environment variable not set');
  process.exit(1);
}

const bot = new Telegraf(TELEGRAM_BOT_TOKEN);

/**
 * Fetch custom emoji sticker data from Telegram
 */
async function getCustomEmojiSticker(customEmojiId) {
  const url = `https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/getCustomEmojiStickers`;
  const response = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ custom_emoji_ids: [customEmojiId] })
  });

  const data = await response.json();
  if (!data.ok || !data.result || data.result.length === 0) {
    throw new Error('Custom emoji not found');
  }

  return data.result[0];
}

/**
 * Download file from Telegram
 */
async function downloadFile(fileId) {
  const fileUrl = `https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/getFile?file_id=${fileId}`;
  const fileResponse = await fetch(fileUrl);
  const fileData = await fileResponse.json();

  if (!fileData.ok) {
    throw new Error('Failed to get file path');
  }

  const downloadUrl = `https://api.telegram.org/file/bot${TELEGRAM_BOT_TOKEN}/${fileData.result.file_path}`;
  const downloadResponse = await fetch(downloadUrl);
  const buffer = await downloadResponse.buffer();

  return buffer.toString('base64');
}

/**
 * Convert sticker to WEBP based on type
 */
async function convertStickerToWebp(sticker) {
  const fileId = sticker.file_id;
  const base64Data = await downloadFile(fileId);

  // Determine type by checking sticker properties
  if (sticker.is_animated) {
    // TGS format (Lottie)
    return await tgsToWebp(base64Data);
  } else if (sticker.is_video) {
    // WEBM format
    return await webmToWebp(base64Data);
  } else {
    // WEBP format (static or animated)
    return await webpToWebp(base64Data);
  }
}

/**
 * Process custom emoji and generate WEBP
 */
async function processCustomEmoji(customEmojiId) {
  // Check cache first
  const cached = cache.get(customEmojiId);
  if (cached) {
    console.log(`Cache hit for ${customEmojiId}`);
    return cached.webp_data;
  }

  console.log(`Cache miss for ${customEmojiId}, generating...`);

  // Fetch sticker data
  const sticker = await getCustomEmojiSticker(customEmojiId);

  // Convert to WEBP
  const webpBuffer = await convertStickerToWebp(sticker);

  // Cache the result
  cache.set(customEmojiId, webpBuffer, 'image/webp');

  return webpBuffer;
}

// Telegram bot message handler
bot.on('message', async (ctx) => {
  const entities = ctx.message.entities || [];

  // Find all custom emoji entities
  const customEmojiIds = entities
    .filter(e => e.type === 'custom_emoji' && e.custom_emoji_id)
    .map(e => e.custom_emoji_id);

  if (customEmojiIds.length === 0) {
    return ctx.reply('Send me custom emoji to get markdown codes!');
  }

  try {
    // Generate WEBPs for all custom emojis (in background)
    const promises = customEmojiIds.map(id => processCustomEmoji(id).catch(err => {
      console.error(`Failed to process ${id}:`, err);
      return null;
    }));

    await Promise.all(promises);

    // Generate markdown codes
    const markdownCodes = customEmojiIds.map(id => `![emoji](${SERVER_URL}/${id}.webp)`);

    const response = `Obsidian markdown:\n\n${markdownCodes.join('\n')}\n\nTemplater snippet:\n\`\`\`\n${markdownCodes.join('\n')}\n\`\`\``;

    await ctx.reply(response);
  } catch (error) {
    console.error('Error processing custom emoji:', error);
    await ctx.reply('Error processing custom emoji. Please try again.');
  }
});

// Express server
app.use(express.json());

app.get('/:id.webp', async (req, res) => {
  const emojiId = req.params.id;

  try {
    // Check cache
    let cached = cache.get(emojiId);

    if (!cached) {
      // Generate on-demand
      console.log(`On-demand generation for ${emojiId}`);
      const webpBuffer = await processCustomEmoji(emojiId);
      cached = { webp_data: webpBuffer, mime_type: 'image/webp' };
    }

    res.setHeader('Content-Type', cached.mime_type);
    res.setHeader('Cache-Control', 'public, max-age=31536000'); // 1 year
    res.send(cached.webp_data);
  } catch (error) {
    console.error(`Error serving ${emojiId}:`, error);
    res.status(404).json({ error: 'Emoji not found' });
  }
});

app.get('/health', (req, res) => {
  const stats = cache.stats();
  res.json({ ok: true, cache: stats });
});

// Telegram webhook endpoint
const WEBHOOK_PATH = `/${TELEGRAM_BOT_TOKEN}`;
app.post(WEBHOOK_PATH, (req, res) => {
  bot.handleUpdate(req.body, res);
});

// Start services
const PORT = process.env.PORT || 3000;

app.listen(PORT, async () => {
  console.log(`Express server running on :${PORT}`);

  // Set webhook
  const webhookUrl = `${SERVER_URL}${WEBHOOK_PATH}`;
  try {
    await bot.telegram.setWebhook(webhookUrl);
    console.log(`Webhook set to: ${webhookUrl}`);
  } catch (error) {
    console.error('Failed to set webhook:', error);
    process.exit(1);
  }
});

// Enable graceful stop
process.once('SIGINT', () => bot.stop('SIGINT'));
process.once('SIGTERM', () => bot.stop('SIGTERM'));
