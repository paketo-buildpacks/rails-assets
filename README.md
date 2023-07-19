# rails-assets

## `gcr.io/paketo-buildpacks/rails-assets`

A Cloud Native Buildpack to precompile rails assets

## Logging Configurations

To configure the level of log output from the **buildpack itself**, set the
`$BP_LOG_LEVEL` environment variable at build time either directly (ex. `pack
build my-app --env BP_LOG_LEVEL=DEBUG`) or through a [`project.toml`
file](https://github.com/buildpacks/spec/blob/main/extensions/project-descriptor.md)
If no value is set, the default value of `INFO` will be used.

The options for this setting are:
- `INFO`: (Default) log information about the progress of the build process
- `DEBUG`: log debugging information about the progress of the build process

```shell
$BP_LOG_LEVEL="DEBUG"
```

## Configuring Exta Assets Directories

By default, the `assets:precompile` command reads assets from a set of specific application paths, such as
`app/assets`, `app/javascript`, `lib/assets` and `vendor/assets`. These directories contain the
source files that need to be precompiled and optimized for production use. The precompiled assets
resulted from running this command are then placed in different directories, such
as `public/assets`, `public/packs` and `tmp/cache/assets`.

Any gem can override the behavior of the `assets:precompile` command, and use different directories
to either read source assets or write the precompilation results. It is possible to set a list of
additional source directories using the `$BP_RAILS_ASSETS_EXTRA_SOURCE_PATHS` environment variable.
In the same way, to set a list of additional destination paths, use `$BP_RAILS_ASSETS_EXTRA_DESTINATION_PATHS`.
Both variables have the same notation of the `$PATH` system variable.

```bash
# adds app/my_gem/assets and lib/other_gem/assets to
# the list of paths containing assets that need precompilation
BP_RAILS_ASSETS_EXTRA_SOURCE_PATHS="app/my_gem/assets:lib/other_gem/assets"

# adds public/my_gem and public/other_gem to
# the list of paths with assets resulting from the
# precompilation process
BP_RAILS_ASSETS_EXTRA_DESTINATION_PATHS="public/my_gem:public/other_gem"
```

Like the `$BP_LOG_LEVEL`, you can set those variables either directly with pack cli or using a `project.toml` file.
