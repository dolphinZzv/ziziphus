# Email Templates

Templates use Go's `text/template` syntax. Filename convention:

```
{template_name}_{lang}.html
```

Supported languages: `zh` (Chinese), `en` (English)
To add a new language, copy an existing file and replace the `{lang}` part with the desired locale code (e.g. `ja`, `ko`, `fr`).

## Available templates

| Template | Description |
|----------|-------------|
| `verify_code` | Verification code email for MFA / email verification |

## Template variables

All templates receive the same data:

| Field | Type | Description |
|-------|------|-------------|
| `Code` | string | The 6-digit verification code |
