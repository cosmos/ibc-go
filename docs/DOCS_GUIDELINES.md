# Documentation guidelines

## Best practices

- Check the spelling and grammar, even if you have to copy and paste from an external source.
- Use simple sentences. Easy-to-read sentences mean the reader can quickly use the guidance you share.
- Try to express your thoughts in a concise and clean way.
- Either Leave a space or use a `-` between the acronyms ADR and ICS and the corresponding number (e.g. ADR 008 or ADR-008, and ICS 27 or ICS-27).
- Don't overuse `code` format when writing in plain English.
- Follow Google developer documentation [style guide](https://developers.google.com/style).
- Check the meaning of words in Microsoft's [A-Z word list and term collections](https://docs.microsoft.com/en-us/style-guide/a-z-word-list-term-collections/term-collections/accessibility-terms) (use the search input!).
- We recommend using RFC keywords in user documentation (lowercase). The RFC keywords are: "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED",  "MAY", and "OPTIONAL. They are to be interpreted as described in [RFC 2119](https://datatracker.ietf.org/doc/html/rfc2119).

### Links

**NOTE:** Strongly consider the existing links (both within this directory and to the website docs) when moving or deleting files.

Relative links should be used nearly everywhere, due to versioning. Note that in case of page reshuffling, you must update all links references.

### Code snippets

Code snippets can be included in the documentation using normal Markdown code blocks. For example:

```md
    ```go
    func() {}
    ```
```

It is also possible to include code snippets from GitHub files by referencing the files directly (and the line numbers if needed). For example:

```md
    ```go reference
    https://github.com/cosmos/ibc-go/blob/v5.0.0/modules/core/04-channel/keeper/handshake.go#L18-L65
    ```
```

## Technical writing course

Google provides a free [course](https://developers.google.com/tech-writing/overview) for technical writing.
