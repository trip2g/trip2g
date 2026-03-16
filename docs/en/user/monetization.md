---
title: Monetization
free: true
home_position: 6
---

Restrict notes to paying subscribers by leaving `free: true` off.

### How access works

Notes without `free: true` are private by default. Only admins and authenticated subscribers can read them. Visitors who are not logged in see a paywall or are redirected to sign in.

This means you can publish two tiers of content in the same vault: public notes with `free: true` and subscriber-only notes without it.

### Patreon integration

<!-- TODO: verify with product — exact Patreon setup steps -->

Connecting Patreon allows your existing Patreon subscribers to authenticate on your site and access subscriber-only notes automatically. Once configured, access is granted based on active subscription status.

### Boosty integration

<!-- TODO: verify with product — exact Boosty setup steps -->

Boosty integration works similarly to Patreon: subscribers authenticate via Boosty and receive access to gated content based on their subscription tier.

### Telegram group access

You can gate content by membership in a Telegram group instead of (or in addition to) a paid subscription:

1. Add your bot to a Telegram group as admin
2. In the admin panel, configure the group under the bot settings
3. The bot checks whether a visitor is a member of that group
4. Members get access to the gated content automatically

This lets you monetize through a paid Telegram group without a third-party payment platform.

### Paywall preview

<!-- TODO: verify with product — confirm whether partial content preview is supported and how many characters are shown -->

Visitors may see a preview of the note before being asked to authenticate. The rest of the content is hidden until they sign in with an active subscription.

### How subscribers authenticate

Subscribers sign in via email (the same magic-link flow used by admins). Once authenticated, the system checks their subscription status with the connected platform (Patreon, Boosty, or Telegram group membership) and grants or denies access accordingly.

### Related

- [[en/user/advanced]] — OAuth and advanced access configuration
