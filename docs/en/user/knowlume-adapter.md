---
free: true
title: Knowlume Adapter
lang_redirect: "[[ru/user/Knowlume Adapter]]"
---

The Knowlume Adapter syncs fragments from [Knowlume](https://knowlu.me) into your trip2g vault on a schedule. Each fragment becomes a Markdown note with frontmatter.

Full documentation is available in Russian: [[ru/user/Knowlume Adapter|Knowlume Adapter (RU)]].

### Quick setup

1. Go to project settings → Scheduled webhooks → Create.
2. Set the webhook URL:

```
https://knowlumegate.2pub.me/webhook?knowlume_api_key=YOUR_KEY
```

3. Set a cron schedule, for example `0 * * * *` for every hour.
4. Save.

`knowlume_api_key` is your Knowlume account API key, found in your profile settings on knowlu.me.

After the first scheduled run, Knowlume fragments appear as notes in your vault.

### Jsonnet filtering

Add a Jsonnet expression to the webhook **instruction** field to filter or transform data before it is applied. `std.extVar("input")` holds the full adapter JSON response.

### Debugger

Open the adapter URL in a browser to access a live debugger with the raw JSON on the left and transformed output on the right.

See also: [[en/user/webhooks|Webhooks & automation]].
