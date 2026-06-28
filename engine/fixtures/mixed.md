# Mixed Content Test

This fixture combines all Markdown elements to test the engine handles mixed content correctly.

## Section 1: Text and Lists

Welcome to the mixed content test. This paragraph has **bold**, *italic*, ***both***, ~~strike~~, `code`, and [a link](https://example.com).

### Shopping List

- **Dairy**
  - Milk
  - Cheese *(cheddar)*
  - Yogurt
- **Produce**
  - Apples
  - Bananas
  - ~~Kale~~ (out of stock)
- **Bakery**
  - Whole wheat bread
  - ~~Croissants~~

### Steps to Complete

1. **Prepare** the workspace
   1. Clean desk
   1. Gather tools
1. **Execute** the plan
   - Step A
   - Step B
1. **Review** results

## Section 2: Code and Quotes

> **Tip:** Always test your code before deploying to production.
>
> `console.log("Hello, world!");`

```go
package main

import "fmt"

func main() {
    fmt.Println("Go code block in mixed content")
}
```

## Section 3: Tables

| Feature | Status | Priority |
|---------|--------|----------|
| Bold | Done | High |
| Italic | Done | High |
| Tables | Done | High |
| Images | Done | Medium |

---

## Section 4: Edge Cases

### Code block right after heading

```
code immediately after heading
```

### Text right after code block

Text right after code block without extra spacing.

### Empty line handling

This paragraph is followed by an empty line.

This is the paragraph after the empty line.

### Very short content

Hi.
