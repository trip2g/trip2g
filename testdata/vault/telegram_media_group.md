---
telegram_publish_at: 2025-11-18T09:38:00
telegram_publish_tags:
  - test_channel
free: true
title: Media Group Telegram Post
---

This post contains **multiple media files** (2-10) and will be sent using `sendMediaGroup` API method.

The post type is: **media_group**

![[telegram_photo.png]]
![[telegram_photo2.jpg]]
![[telegram_video.mp4]]

Features:
- Multiple photos and videos (up to 10)
- Only first media gets the caption
- Caption can be edited with `editMessageCaption`
- Media files cannot be changed after sending

This tests media group functionality with mixed photo and video content.
