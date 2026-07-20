---
free: true
title: On your phone
---

The sync plugin runs on Obsidian mobile. It is not in the community plugin store, so you do not install it there — you put *this vault* on the phone, plugin and all.

### iPhone and iPad

1. **Download the archive again, on the phone.** In the phone's browser, signed in as admin, open `{{publicUrl}}/_system/onboarding-vault`. It downloads straight away.
2. **Tap the file.** iOS unpacks it and leaves a folder called `vault`.
3. **Move it out of Downloads.** Long-press, move it to a folder like `Obsidian` under *On My iPhone*. Rename it to whatever you want the vault called — nothing depends on the name.
4. **Open it in Obsidian** → **Manage vaults**. The folder is in the list already.
5. **Sync** from the burger menu.

Nothing to configure: the instance URL and API key travel inside the archive.

### Android

Same five steps. Any file manager unpacks the archive, and Obsidian's vault picker asks for folder access the first time.

### Why you cannot just copy the plugin across

Plugins live in `.obsidian/plugins/`, and iOS hides dot-folders in the Files app — you cannot navigate in there to drop a plugin. Arriving *inside* an archive sidesteps the problem entirely.

> [!note] Each download mints a new API key
> Downloading the vault a second time for the phone is expected and fine. Old keys stay valid until you remove them in the admin panel under **API Keys** — tidy up the ones you are not using.

### Updating the plugin later

The vault also ships [BRAT](https://tfthacker.com/BRAT), which installs plugins from GitHub. Since trip2g is not in the community store, BRAT is how you pick up a new plugin version on a phone without downloading a whole new vault. Open BRAT's settings and check for updates.

### If sync fails

A network error mentioning `Access-Control-Allow-Origin` means your instance predates the fix that allows Obsidian mobile's origins (`capacitor://localhost` on iOS, `http://localhost` on Android). Update the instance.

Full guide: [Obsidian on a phone](https://trip2g.com/en/user/mobile).

### Next

[[what-else]] — everything else the platform does.
