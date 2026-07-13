#!/usr/bin/env python3
"""Real-Telegram-client driver for the TG<->subgraph e2e test.

This script drives a REAL, authenticated Telegram user account by reusing the
telegram-mcp server's own tool functions (send_message, get_messages,
join_chat_by_link, leave_chat, ...). It imports telegram-mcp's `main` module and
calls the exact async functions the MCP server exposes over stdio, so the e2e
exercises the same code path a real MCP client would. The account is
authenticated non-interactively from telegram-mcp's pre-generated
TELEGRAM_SESSION_STRING (Telethon StringSession) in its `.env`.

The Go e2e test (e2e_bidirectional_access_test.go) shells out to this driver.
It is invoked as small, single-purpose subcommands and prints machine-readable
output on stdout; a nonzero exit code signals failure.

Env:
  TG_E2E_MCP_DIR   directory of the telegram-mcp checkout (default: ~/projects/telegram-mcp).
                   Its `.env` must define TELEGRAM_API_ID/HASH/SESSION_STRING.

Subcommands:
  whoami                              print the connected account id (connectivity check)
  send    --chat ID    --text T       send a message
  messages --chat ID  [--limit N]     print recent messages (newest first)
  buttons --chat ID   [--limit N]     print inline-button labels on the latest keyboard message
  join    --link URL                  join a group/channel by invite link
  leave   --chat ID                   leave a group/channel
"""

import argparse
import asyncio
import os
import sys
from pathlib import Path


def _load_mcp():
    mcp_dir = Path(
        os.environ.get("TG_E2E_MCP_DIR", str(Path.home() / "projects" / "telegram-mcp"))
    ).expanduser()
    env_file = mcp_dir / ".env"
    if not env_file.exists():
        print(f"telegram-mcp .env not found at {env_file}", file=sys.stderr)
        sys.exit(3)
    # telegram-mcp's main.py reads its .env via load_dotenv() from cwd and opens
    # a StringSession, so run from the checkout dir.
    os.chdir(mcp_dir)
    sys.path.insert(0, str(mcp_dir))
    import main  # noqa: E402  (telegram-mcp module)

    return main


async def _run(args):
    main = _load_mcp()
    await main.client.connect()
    if not await main.client.is_user_authorized():
        print("session not authorized (bad/expired TELEGRAM_SESSION_STRING)", file=sys.stderr)
        sys.exit(4)

    try:
        if args.cmd == "whoami":
            me = await main.client.get_me()
            print(me.id)

        elif args.cmd == "send":
            out = await main.send_message(chat_id=args.chat, message=args.text)
            print(out)
            if "successfully" not in out.lower():
                sys.exit(5)

        elif args.cmd == "messages":
            out = await main.get_messages(chat_id=args.chat, page=1, page_size=args.limit)
            print(out)

        elif args.cmd == "buttons":
            out = await main.list_inline_buttons(chat_id=args.chat, limit=args.limit)
            print(out)

        elif args.cmd == "join":
            out = await main.join_chat_by_link(link=args.link)
            print(out)

        elif args.cmd == "leave":
            out = await main.leave_chat(chat_id=args.chat)
            print(out)

        else:
            print(f"unknown command {args.cmd}", file=sys.stderr)
            sys.exit(2)
    finally:
        await main.client.disconnect()


def main_cli():
    p = argparse.ArgumentParser(description=__doc__)
    sub = p.add_subparsers(dest="cmd", required=True)

    sub.add_parser("whoami")

    s = sub.add_parser("send")
    s.add_argument("--chat", required=True)
    s.add_argument("--text", required=True)

    m = sub.add_parser("messages")
    m.add_argument("--chat", required=True)
    m.add_argument("--limit", type=int, default=10)

    b = sub.add_parser("buttons")
    b.add_argument("--chat", required=True)
    b.add_argument("--limit", type=int, default=20)

    j = sub.add_parser("join")
    j.add_argument("--link", required=True)

    lv = sub.add_parser("leave")
    lv.add_argument("--chat", required=True)

    args = p.parse_args()

    import nest_asyncio

    nest_asyncio.apply()
    asyncio.run(_run(args))


if __name__ == "__main__":
    main_cli()
