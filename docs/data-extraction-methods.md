# Data extraction methods, and the security model that has to wrap them

Research date: 2026-10-27. Status: findings for discussion. This answers two
questions together because they're actually the same question: *how* you're
allowed to read a category of data off a phone determines *what* the
security model for that category has to look like — there's no
filesystem-level shortcut left on either platform, so the extraction method
and the permission/trust boundary are the same design decision.

This applies specifically to the **selective extraction / structured
dataset** product direction (`product-expansion.md` §1, §5), not the
VM-persona mode — a persona VM doesn't "extract" anything, apps running
inside it just write their data there natively. The question below only
matters for pulling existing on-device data (Photos, Contacts, Documents...)
into a category dataset.

## The old approach is dead; there's no bypass left

`adb backup` — historically the way to pull a full, unencrypted app-data
backup off an Android device for forensics or migration tooling — was
deprecated in Android 12 and the binary itself was removed from
platform-tools entirely as of release r34 (May 2023). Since Android 12, even
`android:allowBackup="true"` no longer grants `adb` access; many apps now
explicitly set `android:allowBackup="false"` or use the newer
`dataExtractionRules` to block it outright. [securitycafe.ro][adb-dead]
[nirajkharel][adb-extraction]

**Read that as confirmation, not an obstacle to route around.** It means
every legitimate data source for extraction is a scoped, user-consented API
call through the platform's own permission system — which is good, because
it means the extraction method and the security model aren't two separate
design problems. Request exactly the permission for the category the user
opted into, nothing broader, and the OS enforces the boundary for you.

## Android: content providers, gated by the OS's own per-category permissions

| Category | Mechanism | Permission | Feasibility |
|---|---|---|---|
| Photos/Video | `MediaStore` | `READ_MEDIA_IMAGES` / `READ_MEDIA_VIDEO` (Android 13+ granular media permissions) | Straightforward, standard consent prompt |
| Contacts | `ContactsContract` | `READ_CONTACTS` | Straightforward, though Google rolled out a dedicated, stricter Contacts Permission policy in April 2026 — re-check current Play Console requirements before relying on this |
| Documents/Files | Storage Access Framework (user picks a file or folder explicitly) | None beyond the user's picker interaction; grant persists via `takePersistableUriPermission` | Works, but is inherently user-driven per grant, not bulk/automatic — fits a one-time "pick your Documents folder" flow fine |
| SMS / Call Log | `Telephony` provider / `CallLog` | `READ_SMS`, `READ_CALL_LOG` | **Only available to the device's default SMS/Phone/Assistant handler app.** Google manually reviews and approves this use case in Play Console — this is a real product/launch-timeline decision, not a checkbox. [Play Console sensitive permissions][play-sensitive] [XDA][xda-sms] |
| Any other app's private data (WhatsApp, Signal, banking apps, etc.) | None | N/A | **Not accessible, full stop**, under scoped storage, on any Android version currently shipping. The only way ShadowDevice ever touches this data is by the app running *inside* the persona VM and writing it there itself — there is no extraction path for it. |

[scoped storage overview][scoped-storage] [content provider basics][content-providers]

The SMS/Call Log row deserves its own callout: becoming a phone's default
SMS handler just to back up text messages is a large ask (it changes how the
user sends texts day-to-day, not just a permission grant), and Google
manually gates it. Worth questioning whether "Messages" as a selectable
category should mean plain SMS/MMS at all — most people's actual messaging
happens in WhatsApp/iMessage/Signal today, none of which are reachable this
way regardless (see the private-data row above). Those apps need their own,
app-specific integration (e.g., WhatsApp's own linked-device/export
mechanism, as already noted in the original PDF) — a different problem per
app, not a generalizable "extract files from the device" method.

## iOS: per-category frameworks with explicit consent, and one hard wall

| Category | Mechanism | Feasibility |
|---|---|---|
| Photos | `PHPhotoLibrary`/`PhotoKit` | Standard consent prompt |
| Contacts | `Contacts` framework | Standard consent prompt |
| Calendar | `EventKit` | Standard consent prompt |
| Documents | Document picker / Files app integration | User-driven per grant, same shape as Android's SAF |
| Messages / iMessage / SMS | **None — no public API exists.** | All message content lives in a local SQLite database (`~/Library/Messages/chat.db`) that is categorically inaccessible to any third-party app on iOS. This isn't a restricted permission like Android's SMS handler requirement — there is no API to request at all, on any iOS version. Confirmed across Apple's own developer forum threads and independent technical writeups. [confirmed][imessage-no-api] |
| Any other app's private data | None | Same wall as Android — sandboxing blocks it entirely, and iOS enforces this more strictly, with no equivalent to "become the default handler" for any category. |

So on iOS, "Messages" isn't a smaller version of the Android feature — it's
not a feature that can exist at all through any sanctioned API. If this
category matters to the product, the iOS story has to be framed honestly as
"not available on iOS" rather than "limited," and that's worth deciding
early since it shapes what the per-category picker (`product-expansion.md`
§1) can even offer per platform.

## A third path worth considering: cloud-side pull instead of on-device extraction

For data that's already synced to a cloud account the user controls — Google
Photos, Gmail, Google Drive — Google's **Data Portability API**
(`developers.google.com/data-portability`, built on the cross-industry **Data
Transfer Project** Google co-founded) is a sanctioned, OAuth-consented way to
pull a copy of that data directly from Google's servers, with no on-device
extraction step at all. [Data Transfer Project][dtp] [Data Portability
API][takeout-changes] This is a genuinely different ingestion path — instead
of "read this off the phone," it's "pull this from the account it's already
backed up to" — and could be the better mechanism for categories like Photos
when the user already has cloud backup enabled, leaving on-device extraction
APIs for data that never left the device.

## Security implementation, specifically for this extraction pipeline

This is the part of "how do we thoroughly implement security" that's
actually specific to extraction (the broader trust model is already in
`design-review.md`/`SECURITY.md`):

1. **Least privilege, enforced by construction, not by policy.** Request
   only the runtime permission for the category the user actually opted
   into in the picker — never a broad grant like
   `MANAGE_EXTERNAL_STORAGE`. This isn't just good practice; broad storage
   permissions draw specific extra Play Store review scrutiny, so
   over-asking is a launch-risk problem, not only a trust problem.
2. **Nothing persists unencrypted, even transiently.** Read from the content
   provider/photo library straight into memory, seal it with the
   category-disk's block key (`internal/block`), and never let a plaintext
   copy touch the app's own private storage — even as a cache file —
   before encryption. This is the same discipline the block-store code
   already follows (RAM-only until sealed); extraction has to inherit it,
   not introduce a new "temp file on the way to encryption" exception.
3. **SAF/document-picker grants are capability tokens, not cached content.**
   Persist the opaque URI permission so the user isn't re-prompted every
   sync, but never cache the file content itself outside of the
   encrypt-then-store pipeline — the grant should only ever be used to
   re-read-and-reseal on each sync, never to keep a plaintext copy around
   "for convenience."
4. **The default-SMS-handler path needs its own threat model if pursued.**
   Becoming the default SMS app means the product now also intercepts and
   can send messages — a materially larger trust surface than read-only
   extraction, and Google's manual review process means it's also a
   material launch-timeline dependency. Treat it as a separate, explicitly
   scoped feature decision, not a natural extension of "add one more
   content provider."

## Sources

[adb-dead]: https://securitycafe.ro/2026/02/02/mobile-pentesting-101-the-death-of-adb-backup-modern-data-extraction-in-2026/
[adb-extraction]: https://nirajkharel.com.np/posts/allow-backup-data-extraction/
[scoped-storage]: https://developer.android.com/training/data-storage#scoped-storage
[content-providers]: https://developer.android.com/guide/topics/providers/content-provider-basics
[play-sensitive]: https://support.google.com/googleplay/android-developer/answer/16558241?hl=en
[xda-sms]: https://www.xda-developers.com/google-restriction-sms-call-log-permissions/
[imessage-no-api]: https://fatbobman.com/en/posts/deep-dive-into-imessage/
[dtp]: https://en.wikipedia.org/wiki/Data_Transfer_Project
[takeout-changes]: https://www.digitalinformationworld.com/2023/08/google-makes-key-changes-to-takeout.html
