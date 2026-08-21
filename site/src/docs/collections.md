# collections

A collection is a directory of YAML files; subdirectories become folders. A root-level
`config/config.yaml` marks a directory as a collection:

```yaml
version: 1
```

Open any directory with `-dir` (or the current directory) and lazypost initializes it automatically.
Legacy `.lazypost` markers remain readable for the current session and migrate to
`config/config.yaml` on the first write. `./sample-collections` / `./collections` are implicit
collections without a marker.

Collection writes are root-confined and atomic. New requests, folders, environments, and renames
refuse path collisions instead of replacing existing data; newly created request and environment
files default to owner-only permissions.

## request format

```yaml
name: create post
method: POST
url: "{{host}}/posts"
headers:
  - name: Content-Type
    value: application/json
auth:
  type: bearer        # none | basic | bearer | apikey
  token: "{{api_token}}"
body: |
  {"title": "hello"}
```

## auth variants

| kind | yaml |
| --- | --- |
| basic | auth: { type: basic, username: u, password: p } |
| bearer | auth: { type: bearer, token: t } |
| apikey | auth: { type: apikey, keyName: X-Api-Key, keyValue: k, keyIn: header } # or keyIn: query |
