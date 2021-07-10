# hugoext

Utility to parse a hugo config file and create the same file structure for content through an
arbitrary output pipe extension.

Hugo parses primarily markdown files and go templates. The initial motivation for this utility was
to enable the same tools to publish a gemlog version of the same blog to make it accessible through
the Gemini protocol.

**NOTE**: not many features, this is minimal and only has one use case for now.

Features
- reads hugo `.toml` file for section output formats
- supports an arbitrary document processor, any program that supports UNIX pipes

When the selected extension is blank, markdown files will be copied unmodified.

## Example Use

Using the [md2gmi](https://github.com/n0x1m/md2gmi) command line utility to convert markdown to
gemtext. Executed from the hugo directory:

```
hugoext -ext gmi -pipe md2gmi
```

It abides the hugo section config in `[permalinks]` but only uses the content subdirectory to
determine the section. An example section config in hugo looks like this:

```
[permalink]
posts = "/posts/:year/:month:day/:filename"
snippets = "/snippets/:filename"
page = ":filename"
```

### Installation

```
go install github.com/n0x1m/hugoext
```

To use the gemini file server and markdown to gemtext converter in the examples below, also install
these:

```
go install github.com/n0x1m/md2gmi
go install github.com/n0x1m/gmifs
```

### Development

To test the extension in a similar fashion to the hugo workflow, use a server to host the static
files. Here an example for a Gemlog using [gmifs](https://github.com/n0x1m/gmifs) in a makefile:

```makefile
serve:
    hugoext -ext gmi -pipe md2gmi -serve="gmifs -autoindex"
```

hugoext pipes the input through the `md2gmi` extension and spawns `gmifs` to serve the local gemini
directory with auto indexing enabled.

### Production

I have a makefile target in my hugo directory to build and publish html and gemtext content:

```makefile
build:
    hugo --minify
    hugoext -ext gmi -pipe md2gmi

publish: build
    rsync -a -P --delete ./public/ dre@nox.im/var/www/htdocs/nox.im/
```

The output directory for both hugo and hugoext is `./public`. It's ok to mix the two into the same
file tree as each directory will contain an `index.html` and an `index.gmi` file.
