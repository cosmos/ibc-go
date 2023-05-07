TODO: UPDATE ([#3534](https://github.com/cosmos/ibc-go/issues/3534))

# IBC-Go Documentation

Welcome to the IBC-Go documentation! This website is built using [Docusaurus 2](https://docusaurus.io/), a modern static website generator.

## Docs Build Workflow

The documentation for IBC-Go is hosted at <https://ibc.cosmos.network>.

built from the files in this (`/docs`) directory for
[main](https://github.com/cosmos/ibc-go/tree/main/docs).

## docusaurus.config.js

Docusaurus configuration file is located at `./docusaurus.config.js`. This file contains the configuration for the sidebar, navbar, footer, and other settings. Sidebars are created in `./sidebars.js`.

## Links

In docusaurus, there are three ways to link to other pages:

1. File Paths (relative or absolute)
2. URLs (relative or absolute)
3. Hyperlinks

In this section, we will discuss when to use each.

### Multi-Documentation Linking

Technically, there are four docs being maintained in this repo:

1. Found in `docs/docs/` (this is the one displayed on the website in the "Documentation" tab)
2. Found in `docs/architecture/` (this is the one displayed on the website in the "Architecture Decision Records" tab)
3. Found in `docs/events/` (depreciated, this is not displayed on the website, but is hosted under `/events/` url)
4. Found in `docs/params/` (depreciated, this is not displayed on the website, but is hosted under `/params/` url)

When referencing a markdown file, you should use relative file paths if they are in the same docs directory from above. For example, if you are in `docs/docs/01-ibc` and want to link to `docs/docs/02-apps/02-transfer/01-overview.md`, you should use the relative link `../02-apps/02-transfer/01-overview.md`.

If the file you are referencing is in a different docs directory, you should use a absolute URL. For example, if you are in `docs/docs/01-ibc` and want to link to `docs/architecture/adr-001-coin-source-tracing.md`, you should use the absolute URL (not absolute file path), in this case `/architecture/adr-001-coin-source-tracing`. You can find the absolute URL by looking at the slug in the frontmatter of the markdown file you want to link to. If the frontmatter slug is not set (such as in `docs/architecture/adr-001-coin-source-tracing.md`), you should use the url that docusaurus generates for it. You can find this by looking at the url of the page in the browser.

Note that when referencing any file outside of the parent `docs/` directory, you should always use a hyperlink.

### Code Blocks

Code blocks in docusaurus are super-powered, read more about them [here](https://docusaurus.io/docs/markdown-features/code-blocks). Three most important features for us are:

1. We can add a `title` to the code block, which will be displayed above the code block. (This should be used to display the file path of the code block.)
2. We can add a `reference` tag to the code block, which will reference github to create the code block. **You should always use hyperlinks in reference codeblocks.** Here is what a typical code block should look like:

````ignore
```go reference title="modules/apps/transfer/keeper/keeper.go"
https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/apps/transfer/keeper/keeper.go#L19-L31
```
````

3. We can highlight lines in the code block by adding `// highlight-next-line` before the line we want to highlight. We can use this to highlight diffs. Here is an example:

````ignore
```go
import (
  ...
  // highlight-next-line
+ ibctm "github.com/cosmos/ibc-go/v6/modules/light-clients/07-tendermint"
  ...
)
```
````

### Static Assets

Static assets are the non-code files that are directly copied to the build output. They include **images**, stylesheets, favicons, fonts, etc.

By default, you are suggested to put these assets in the `static/` directory. Every file you put into that directory will be copied into the root of the generated build folder with the directory hierarchy preserved. E.g. if you add a file named `sun.jpg` to the static folder, it will be copied to `build/sun.jpg`.

These assets should be referenced using absolute URLs. For example, if you have an image in `static/img/cosmos-logo-bw.png`, you should reference it using `/img/cosmos-logo-bw.png`.

### Raw Assets

If you want to link a raw file, you should link to it using `@site` + its base path. For example, if you want to link to the raw markdown file `/architecture/adr.template.md`, you should use the absolute URL `@site/architecture/adr.template.md`.

## Building Locally

### Installation

```bash
npm install
```

### Local Development

```bash
npm start
```

This command starts a local development server and opens up a browser window. Most changes are reflected live without having to restart the server.

### Build

```bash
npm run build
```

This command generates static content into the `build` directory and can be served using any static contents hosting service.

### Serve

```bash
npm run serve
```

This command starts a local production server and opens up a browser window.

## Search

TODO: update or remove ([#3534](https://github.com/cosmos/ibc-go/issues/3534))

<!-- ## Consistency

Because the build processes are identical (as is the information contained herein), this file should be kept in sync as
much as possible with its [counterpart in the Cosmos SDK repo](https://github.com/cosmos/cosmos-sdk/blob/main/docs/README.md). -->

## Updating the Documentation

The documentation is autogenerated from the markdown files found in [docs](./docs) directory. Each directory in `./docs` represents a category to be displayed in the sidebar. If you create a new directory, you must create a `_category_.json` file in that directory with the following contents:

```json
{
  "label": "Sidebar Label",
  "position": 1, // position of the category in the sidebar
  "link": null
}
```

If you create a new markdown file within a category (`.docs/` directory is itself a category), you must add the following frontmatter to the top of the markdown file:

```yaml
---
title: Title of the file # title of the file in the sidebar
sidebar_label: Sidebar Label # title of the file in the sidebar
sidebar_position: 1 # position of the file in the sidebar
slug: /migrations/v5-to-v6 # the url of the file
---
```

### File and Directory Naming Conventions

Inside `/docs/docs/`:

- All files should be named in `kebab-case`.
- All files should have a two digit prefix, indicating the order in which they should be read and displayed in their respective categories. For example, `01-overview.md` should be read before `02-integration.md`. If this order changes, the prefix should be updated. Note that the ordering is enforced by the frontmatter and not the file name.
- **All files that end in `.template.md` will be ignored by the build process.**
- The prefix `00-` is reserved for root links of categories (if a category has a root link). For example, see [`00-intro.md`](./docs/00-intro.md).
- All category directories should be named in `kebab-case`.
- All category directories must have a `_category_.json` file.
- All category directories should have a two digit prefix (except for the root `./docs` category), indicating the order in which they should be read and displayed in their respective categories. For example, `01-overview.md` should be read before `02-integration.md`. If this order changes, the prefix should be updated. Note that the ordering is enforced by the frontmatter and not the file name.
- The images for each documentation should be kept in the same directory as the markdown file that uses them. This will likely require creating a new directory for each new category. The goal of this is to make versioning easier, discourage repeated use of the image, and make it easier to find images.

## Versioning

Versioning only applies to documentation and not the ADRs found in the `./architecture/` directory.

### Terminology

- Current version: The version placed in the `.docs/` folder. This version is the one that is displayed on the website by default, referred to as next.
- Latest version: This version is defined in `./docusaurus.config.js` file under the `lastVersion` key.

### Overview

A typical versioned doc site looks like below:

```ignore
website
├── sidebars.json          # sidebar for the current docs version
├── docs                   # docs directory for the current docs version
│   ├── 01-foo
│   │   └── 01-bar.md      # https://mysite.com/docs/next/01-foo/01-bar
│   └── 00-intro.md        # https://mysite.com/docs/next/00-intro
├── versions.json          # file to indicate what versions are available
├── versioned_docs
│   ├── version-v1.1.0
│   │   ├── 01-foo
│   │   │   └── 01-bar.md  # https://mysite.com/docs/01-foo/01-bar
│   │   └── 00-intro.md
│   └── version-v1.0.0
│       ├── 01-foo
│       │   └── 01-bar.md  # https://mysite.com/docs/v1.0.0/01-foo/01-bar
│       └── 00-intro.md
├── versioned_sidebars
│   ├── version-v1.1.0-sidebars.json
│   └── version-v1.0.0-sidebars.json
├── docusaurus.config.js
└── package.json
```

The `./versions.json` file is a list of version names, ordered from newest to oldest.

### Tagging a new version

It is possible to tag the current version of the docs as a new version. This will create the appropriate files in `./versioned_docs/` and `./versioned_sidebars/` directories, and modify the `./versions.json` file. To do this, run the following command:

```bash
npm run docusaurus docs:version v7.1.0
```

### Adding a new version

To add a new version:

1. Create a new directory in `./versioned_docs/` called `version-vX.Y.Z` where `X.Y.Z` is the version number. This directory should contain the markdown files for the new version.
2. Create a new file in `./versioned_sidebars/` called `version-vX.Y.Z-sidebars.json`. This file should contain the sidebar for the new version.
3. Add the version to the `./versions.json` file. The list should be ordered from newest to oldest.
4. If needed, make any configuration changes in `./docusaurus.config.js`. For example, updating the `lastVersion` key in `./docusaurus.config.js` to the latest version.

### Updating an existing version

You can update multiple docs versions at the same time because each directory in `./versioned_docs/` represents specific routes when published. Make changes by editing the markdown files in the appropriate version directory.

### Deleting a version

When a version is no longer supported, you can delete it by removing it from `versions.json` and deleting the corresponding files in `./versioned_docs/` and `./versioned_sidebars/`.
