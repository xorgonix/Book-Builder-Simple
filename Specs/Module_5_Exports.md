# Module 5: Manuscript Export Pipeline

## 1. Purpose

The export module turns an approved book prototype into files the author can inspect, archive, and load onto reading devices.

The first export milestone must produce:

- Markdown manuscript source
- Rendered standalone HTML preview
- EPUB 3 ebook suitable for local reader testing and Draft2Digital/Smashwords upload experiments
- JSON lint report

The export system must not become a black box. The user should be able to see which manuscript content and metadata were used to produce each export.

## 2. Export Cleanup Contract

Exports must run a deterministic cleanup pass over the already-generated manuscript. This pass must not regenerate prose or write cleaned prose back into PocketBase.

Cleanup rules:

- Use PocketBase chapter order and chapter title fields as canonical truth.
- Render exactly one canonical heading per chapter.
- Strip duplicate generated chapter headings only from the beginning of chapter prose.
- Normalize Markdown spacing.
- Normalize only obvious simple list patterns.
- Detect large duplicated blocks across chapters and report them.
- Preserve chapter wording outside narrow structural cleanup.

## 3. First-Pass Export Formats

### A. Markdown Manuscript

Markdown is the clean source export.

Required output:

- One `.md` file per book export
- YAML-style front matter
- Title page information
- Table of contents section
- Chapter headings and chapter body text

Markdown should be easy to diff, inspect, archive, and convert later.

### B. Rendered HTML Preview

HTML is the browser reading preview.

Required output:

- One standalone `.html` file per book export
- Title page
- Table of contents with chapter links
- Readable chapter typography
- Export notes for warnings and lint findings that matter to the user

### C. EPUB 3

EPUB is the first reader-feel export.

Required output:

- Valid `.epub` ZIP package
- `mimetype`
- `META-INF/container.xml`
- `OEBPS/content.opf`
- `OEBPS/toc.xhtml`
- `OEBPS/styles.css`
- One XHTML file per chapter

The EPUB must be reflowable, not fixed-layout.

### D. Lint Report

The lint report is the machine-readable export audit.

Required output:

- One `.json` report per book export
- Book id, title, export timestamp, issue count, and issue list
- Issue types for missing prose, fallback source usage, removed headings, front matter injection, and duplicated blocks

## 4. Draft2Digital / Smashwords Compatibility Target

Draft2Digital accepts uploaded EPUB files for ebook distribution and generally preserves author-supplied EPUB formatting. Therefore the first EPUB target should be conservative and standards-based.

EPUB v1 rules:

- Use EPUB 3.
- Use semantic XHTML.
- Use simple CSS.
- Do not embed fonts in v1.
- Do not use fixed positioning.
- Do not use scanned text images as manuscript content.
- Do not force exact page layout.
- Do not rely on blank lines for spacing.
- Put each chapter in a separate XHTML file.
- Generate a proper navigable table of contents.
- Keep image support optional until the image module is designed.

## 5. Source Content Selection

Exports should use the best available chapter text.

Per chapter fallback order:

1. `draft_content`
2. `targeted_rewrite`
3. `raw_draft`

If a chapter has no usable text, the export should include a clear placeholder in Markdown and should either skip the chapter body in EPUB or include a visible placeholder paragraph marked as draft/incomplete.

The export screen must disclose when fallback content was used.

## 6. Required Book Metadata

The export module depends on a publishing metadata block. Until a dedicated metadata collection exists, the app may derive initial values from the project and Book Brief.

Required metadata:

- Title
- Subtitle
- Author name
- Language
- Description / blurb
- Book type: fiction or nonfiction
- Target format profile: `d2d_smashwords_epub`

Optional metadata:

- Publisher
- ISBN
- Copyright statement
- Series name
- Series number
- Keywords
- Categories
- Price
- Publication date
- Cover image path
- Brand colors
- Body font preference
- Heading font preference

Optional chapter metadata:

- Subtitle
- Front matter label
- Front matter blurb

Chapter metadata may be injected into exports only when explicitly present. It must not be inferred from prose during cleanup.

## 7. File Naming

Generated export files should use a stable slug based on the book title.

Example:

```text
exports/
  the-pincer-protocol.md
  the-pincer-protocol.html
  the-pincer-protocol.epub
  reports/the-pincer-protocol-lint.json
```

If a file already exists, create a versioned name rather than overwriting silently.

Example:

```text
exports/
  the-pincer-protocol-v2.epub
```

## 8. Export Record Model

Future implementation should use an `exports` collection.

Suggested fields:

- `project_id`
- `format`: `markdown`, `epub`
- `profile`: `source_markdown`, `d2d_smashwords_epub`
- `status`: `queued`, `running`, `completed`, `failed`
- `source_version`
- `metadata_snapshot`
- `file_path`
- `error_msg`
- `created`
- `updated`

This lets exports be audited after manuscript or metadata changes.

## 9. Routes

Suggested first-pass routes:

```text
POST /api/project/{id}/exports/markdown
POST /api/project/{id}/exports/epub
POST /api/project/{id}/exports/book
GET  /api/project/{id}/exports/{exportId}/status
GET  /api/project/{id}/exports/{exportId}/download
```

`/exports/book` should generate Markdown, rendered HTML, EPUB, and a lint report together.

Long-running export jobs must return HTTP 202 and poll for completion.

## 10. UI Requirements

The export UI should appear after at least one chapter has draftable text.

Required controls:

- Export Markdown
- Export HTML Preview
- Export EPUB
- Export Book Bundle

Required display:

- Last export status
- Export file links
- Source content summary
- Metadata summary
- Warnings for missing author, cover, blurb, or incomplete chapters
- Lint report link

The user must be able to review metadata before generating the EPUB.

## 11. Acceptance Criteria

Markdown export is acceptable when:

- It includes front matter.
- It includes the book title and subtitle.
- It includes all chapters in sort order.
- It uses the fallback order defined above.
- It has one canonical heading per chapter.

HTML preview export is acceptable when:

- The file opens in a browser.
- The table of contents works.
- Chapters appear in sort order.
- Body text is rendered as attractive reading text, not raw Markdown.

EPUB export is acceptable when:

- The file opens in a normal EPUB reader.
- The table of contents works.
- Chapters appear in sort order.
- Body text is readable without custom fonts.
- The EPUB contains valid package files.
- No chapter silently disappears.

Lint report export is acceptable when:

- It is valid JSON.
- It reports fallback source usage.
- It reports missing manuscript text.
- It reports removed duplicate leading headings.
- It reports suspicious duplicated blocks across chapters.

The first version does not need:

- PDF generation
- DOCX generation
- Cover generation
- Image placement
- Print layout
- Advanced typography
