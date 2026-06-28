# Markdown to PDF — Comprehensive Test

This document tests all features. It includes **bold**, *italic*, ***bold+italic***, ~~strikethrough~~, `inline code`, and [links](https://example.com).

## Block Elements

### Paragraphs with formatting

Normal paragraph with **bold text** and *italic text* and ~~strikethrough~~ and `code`.

A paragraph with a [link to example](https://example.com) and **bold inside a [link](https://example.com)**.

### Image

![Test Image](test-image.png)

### Lists

- Item one with **bold**
- Item two with *italic*
- Item three with `code`
  - Nested item A
  - Nested item B with ~~strike~~
    - Deeply nested item

1. First ordered
1. Second ordered with **bold**
1. Third ordered

### Blockquotes

> This is a blockquote with **bold** and *italic* and ~~strikethrough~~.

### Code Blocks

```
Plain code block with no language tag
```

```javascript
function greet(name) {
    return `Hello, ${name}!`;
}
console.log(greet("World"));
```

### Tables

| Name    | Type   | Version | Description                  |
|---------|--------|---------|------------------------------|
| goldmark | Parser | 1.8.2  | Markdown AST parser          |
| gopdf   | Render | 0.36.1  | PDF generation library       |
| Inter   | Font   | 4.1     | UI font family               |

### Thematic Break

---

## Edge Cases

### Long unbroken string

https://this-is-a-very-long-url-that-should-not-overflow-the-page-boundaries.example.com

### Consecutive headings

# H1 Directly Above
## H2 Directly Below
### H3 Directly Below That

### Empty content test

Paragraph with only a single word.
