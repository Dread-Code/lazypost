# $ lazypost import

Converts Postman and Insomnia exports into a lazypost collection without any TUI interaction.

```bash
$ lazypost import postman-collection.json -env postman-dev.json -env postman-prod.json -dir ./collections/my-api
$ lazypost import insomnia-export.yaml -dir ./collections/my-api --dry-run
$ lazypost import insomnia-export-folder/ -dir ./collections/my-api
```

| Flag | Meaning |
| --- | --- |
| -dir <target> | Required. Target collection directory; refuses to touch an existing directory unless --force. |
| -env <file> | Import an environment file (Postman export JSON or Insomnia environment YAML). Repeatable. |
| --format postman\|insomnia | Override automatic format detection. |
| --dry-run | Parse, validate, and print what would be imported — never writes. Also never creates the target directory. |
| --force | Replace an existing target: moved aside and removed only after the new tree is in place. |
| --strict | Fail the import when any warning is produced. |

## behavior

- Sources: Postman Collection v2.1 JSON, Insomnia v4 JSON (`__export_format: 4`), and Insomnia v5 YAML — a single file or a full export directory.
- Format detection is automatic from file contents; `--format` only for ambiguous inputs.
- Workspaces become top-level folders; Insomnia export directories combine all workspaces, skipping unrelated resources (mock servers, OpenAPI documents) with warnings.
- Collection/base variables become a `base` environment plus one per named environment; multi-workspace imports namespace them logically as `workspace--environment` (slugged filenames use `workspace-environment`); unscoped directory environments use `shared-environment` with a warning. Insomnia's `{{ _.var }}` placeholders normalize to `{{var}}`.
- Structured query parameters become the request's canonical `query` list, so raw URL queries are not sent twice; intentional repeated values remain supported.
- Everything validates first, stages in a temporary sibling directory, then renames into place; workspace, environment, and request filename collisions get deterministic suffixes with warnings; a `config/config.yaml` marker is created automatically.
- Unsupported features (JS pre/test scripts, multipart/binary/GraphQL bodies, unsupported auth) are reported per request and omitted — never guessed. `--strict` promotes any warning to a failure.
