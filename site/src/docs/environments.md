# environments

Put YAML files in `<collection>/environments/`:

```yaml
# environments/dev.yaml
variables:
  host: https://api.dev.example.com
  api_token: secret
```

Select an environment with `ctrl+e`; `{{host}}`-style placeholders are substituted at send time.
Unknown placeholders are left as-is — except in the URL, where an unresolved placeholder fails
loudly and points you at `ctrl+e`.

## the environment manager

`ctrl+/` opens the command palette: `Environments` opens the environment manager — a tab bar of
environments (`ctrl+e` cycles tabs, `a`/`r`/`d` add/edit/delete `key=value` variables, `enter`
activates the tab's env). A leading `/` in the add-variable prompt creates a new empty environment
instead.
