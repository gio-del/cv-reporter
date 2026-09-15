---
title: LAN-reachable mode
description: An opt-in way to reach your Sumisura from another device on a network you trust — and the trade-offs that come with it.
---

By default Sumisura binds to `127.0.0.1` and has no authentication. That is the
design (nothing but your own machine can reach it), not an oversight.

LAN mode is an explicit exception, for checking your applications from a phone
on your own home network.

## Turning it on

In `.env`:

```sh
BIND_ADDR=0.0.0.0
LAN_AUTH_TOKEN=<a long random string>
```

Then start with the `lan` profile:

```sh
docker compose --profile lan up
```

Every `/api/*` request must now carry the token:

```
X-Sumisura-Token: <your token>
```

Requests without it get `401`. Leaving `LAN_AUTH_TOKEN` unset skips the check
entirely, so a plain `docker compose up` behaves exactly as before.

## Know what you are turning on

:::caution
This is a single static shared secret, not a login system, and there is **no
TLS**. The token travels in plaintext across your network. Turn this on only on
a network you control and trust, and treat it as "my phone can reach my laptop",
not "this is exposed safely".
:::

There is also no UI yet for storing the token per device — attach the header
from whatever client you use.

If you need real remote access, put it behind a VPN (Tailscale, WireGuard) and
leave LAN mode off.
